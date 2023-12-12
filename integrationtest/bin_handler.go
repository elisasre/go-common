package integrationtest

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

type BinHandler struct {
	buildCmd *exec.Cmd
	runCmd   *exec.Cmd

	base      string
	target    string
	bin       string
	buildArgs []string
	buildEnv  []string
	runArgs   []string
	runEnv    []string
	coverDir  string
}

func (bh *BinHandler) Build() error {
	if bh.bin == "" {
		parts := strings.Split(strings.TrimSuffix(bh.target, "/"), "/")
		name := parts[len(parts)-1]
		bh.bin = path.Join(BinDir, name)
	}

	pkgs, err := ListPackages(bh.base, "./...")
	if err != nil {
		return fmt.Errorf("listing packages failed: %w", err)
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
	return bh.buildCmd.Run()
}

func (bh *BinHandler) Start() error {
	err := os.MkdirAll(path.Join(bh.base, bh.coverDir), 0o755)
	if err != nil {
		return err
	}

	bh.runEnv = append(bh.runEnv, "GOCOVERDIR="+bh.coverDir)
	bh.runCmd = exec.Command(bh.bin, bh.runArgs...) //nolint:gosec
	bh.runCmd.Stdout = os.Stdout
	bh.runCmd.Stderr = os.Stderr
	bh.runCmd.Env = append(bh.runCmd.Environ(), bh.runEnv...)
	bh.runCmd.Dir = bh.base

	fmt.Println("PWD:", bh.runCmd.Dir)
	fmt.Println("CMD:", bh.runCmd.String())
	if err := bh.runCmd.Start(); err != nil {
		return err
	}

	return nil
}

func (bh *BinHandler) Stop() error {
	if err := bh.runCmd.Process.Signal(os.Interrupt); err != nil {
		return err
	}
	return bh.runCmd.Wait()
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
	pkgs := strings.Split(strings.ReplaceAll(string(pkgsRaw), "\r\n", ","), "\n")
	return pkgs, nil
}
