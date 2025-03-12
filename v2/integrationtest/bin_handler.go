package integrationtest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

type BinHandler struct {
	buildCmd *exec.Cmd
	runCmd   *exec.Cmd

	buildOnce *sync.Once
	buildErr  error

	base          string
	target        string
	bin           string
	buildArgs     []string
	buildEnv      []string
	runArgs       []string
	runEnv        []string
	runStdout     io.Writer
	runStderr     io.Writer
	runInheritEnv bool
	coverDir      string
	opts          []BinOpt
}

func NewBinHandler(opts ...BinOpt) *BinHandler {
	return &BinHandler{
		base:          ".",
		opts:          opts,
		buildOnce:     &sync.Once{},
		runStdout:     os.Stdout,
		runStderr:     os.Stderr,
		runInheritEnv: true,
	}
}

// AddOpts adds options to bin handler.
func (bh *BinHandler) AddOpts(opts ...BinOpt) {
	bh.opts = append(bh.opts, opts...)
}

// Init applies all options to bin handler.
func (bh *BinHandler) Init() error {
	for _, opt := range bh.opts {
		if err := opt(bh); err != nil {
			return err
		}
	}

	// If no base path was provided let's apply proper checks for default path.
	if bh.base == "." {
		if err := BinOptBase(bh.base)(bh); err != nil {
			return err
		}
	}
	return nil
}

func (bh *BinHandler) Build() error {
	bh.buildOnce.Do(func() {
		if bh.bin == "" {
			parts := strings.Split(strings.TrimSuffix(bh.target, "/"), "/")
			name := parts[len(parts)-1]
			bh.bin = path.Join(BinDir, name)
		}

		pkgs, err := ListPackages(bh.base, "./...")
		if err != nil {
			bh.buildErr = fmt.Errorf("listing packages failed: %w", err)
			return
		}

		if len(bh.buildArgs) == 0 {
			coverPkgs := "-coverpkg=" + strings.Join(pkgs, ",")
			bh.buildArgs = []string{"-race", "-cover", "-covermode", "atomic", coverPkgs}
		}

		bh.buildArgs = append([]string{"build", "-o", bh.bin}, bh.buildArgs...)
		bh.buildArgs = append(bh.buildArgs, bh.target)
		bh.buildEnv = append(bh.buildEnv, "CGO_ENABLED=1")

		bh.buildCmd = exec.Command("go", bh.buildArgs...) //nolint:gosec
		bh.buildCmd.Stdout = os.Stdout
		bh.buildCmd.Stderr = os.Stderr
		bh.buildCmd.Env = append(bh.buildCmd.Environ(), bh.buildEnv...)
		bh.buildCmd.Dir = bh.base
		fmt.Println("PWD:", bh.buildCmd.Dir)
		fmt.Println("CMD:", bh.buildCmd.String())
		bh.buildErr = bh.buildCmd.Run()
	})
	return bh.buildErr
}

func (bh *BinHandler) initRunCommand() error {
	err := os.MkdirAll(path.Join(bh.base, bh.coverDir), 0o755)
	if err != nil {
		return err
	}

	bh.runEnv = append(bh.runEnv, "GOCOVERDIR="+bh.coverDir)
	bh.runCmd = exec.Command(bh.bin, bh.runArgs...) //nolint:gosec
	bh.runCmd.Stdout = bh.runStdout
	bh.runCmd.Stderr = bh.runStderr
	if bh.runInheritEnv {
		bh.runCmd.Env = append(bh.runCmd.Environ(), bh.runEnv...)
	} else {
		bh.runCmd.Env = bh.runEnv
	}
	bh.runCmd.Dir = bh.base

	fmt.Println("PWD:", bh.runCmd.Dir)
	fmt.Println("CMD:", bh.runCmd.String())
	return nil
}

func (bh *BinHandler) Start() error {
	if err := bh.initRunCommand(); err != nil {
		return err
	}
	return bh.runCmd.Start()
}

func (bh *BinHandler) Run() error {
	if err := bh.initRunCommand(); err != nil {
		return err
	}
	return bh.runCmd.Run()
}

func (bh *BinHandler) Stop() error {
	if err := bh.runCmd.Process.Signal(os.Interrupt); err != nil {
		return err
	}
	return bh.runCmd.Wait()
}

func (bh BinHandler) Copy() *BinHandler {
	return &bh
}

func ListPackages(base, target string) ([]string, error) {
	cmd := exec.Command("go", "list", target)
	cmd.Dir = base
	cmd.Stderr = os.Stderr
	fmt.Println("PWD:", cmd.Dir)
	fmt.Println("CMD:", cmd.String())
	pkgsRaw, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	pkgs := make([]string, 0)
	lines := bufio.NewScanner(bytes.NewReader(pkgsRaw))
	for lines.Scan() {
		pkg := strings.TrimSpace(lines.Text())
		if pkg != "" {
			pkgs = append(pkgs, pkg)
		}
	}
	return pkgs, nil
}
