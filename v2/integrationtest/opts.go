package integrationtest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go/modules/compose"
)

// Opt is option type for IntegrationTestRunner.
type Opt func(*IntegrationTestRunner) error

// OptBase sets execution base path IntegrationTestRunner.
// OptBase should be usually the first option when passing options to NewIntegrationTestRunner.
func OptBase(base string) Opt {
	return func(itr *IntegrationTestRunner) error {
		absBase, err := filepath.Abs(base)
		if err != nil {
			return fmt.Errorf("getting absolute path for base: '%s' failed: %w", base, err)
		}

		itr.base = absBase
		itr.binHandler.base = absBase
		return nil
	}
}

// OptBinHandler allows setting BinHandler.
// This can be useful when you want to reuse BinHandler between multiple IntegrationTestRunners.
func OptBinHandler(bh *BinHandler) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.binHandler = bh
		return nil
	}
}

// OptTarget sets path to compilation target.
func OptTarget(target string) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptTarget(target)(itr.binHandler)
	}
}

// OptOutput sets output for compilation target.
func OptOutput(output string) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptOutput(output)(itr.binHandler)
	}
}

// OptRunArgs adds args to run arguments for test binary.
func OptRunArgs(args ...string) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptRunArgs(args...)(itr.binHandler)
	}
}

// OptBuildArgs adds args to build arguments for test binary.
func OptBuildArgs(args ...string) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptBuildArgs(args...)(itr.binHandler)
	}
}

// OptRunEnv adds env to test binary's run env.
func OptRunEnv(env ...string) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptRunEnv(env...)(itr.binHandler)
	}
}

// OptBuildEnv adds env to test binary's build env.
func OptBuildEnv(env ...string) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptBuildEnv(env...)(itr.binHandler)
	}
}

// OptCoverDir sets coverage directory for test binary.
func OptCoverDir(coverDir string) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptCoverDir(coverDir)(itr.binHandler)
	}
}

// OptRunStdout sets stdout for test binary.
func OptRunStdout(stdout io.Writer) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptRunStdout(stdout)(itr.binHandler)
	}
}

// OptRunStderr sets stderr for test binary.
func OptRunStderr(stderr io.Writer) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptRunStderr(stderr)(itr.binHandler)
	}
}

// OptRunInheritEnv sets boolean to determine whether test binary inherits env from caller.
func OptRunInheritEnv(inherit bool) Opt {
	return func(itr *IntegrationTestRunner) error {
		return BinOptRunInheritEnv(inherit)(itr.binHandler)
	}
}

// OptTestMain allows wrapping testing.M into IntegrationTestRunner.
// Example TestMain:
//
//	import (
//		"io"
//		"log"
//		"os"
//		"time"
//
//		it "github.com/elisasre/go-common/v2/integrationtest"
//		tc "github.com/testcontainers/testcontainers-go/modules/compose"
//	)
//
//	func TestMain(m *testing.M) {
//		os.Setenv("GOFLAGS", "-tags=integration")
//		itr := it.NewIntegrationTestRunner(
//			it.OptBase("../"),
//			it.OptTarget("./cmd/app"),
//			it.OptCoverDir(it.IntegrationTestCoverDir),
//			it.OptCompose("docker-compose.yaml", it.ComposeUpOptions(tc.Wait(true))),
//			it.OptWaitHTTPReady("http://127.0.0.1:8080/healthz", time.Second*10),
//			it.OptTestMain(m),
//		)
//		if err := itr.InitAndRun(); err != nil {
//			log.Fatal(err)
//		}
//	}
//
// Before using this pattern be sure to read how TestMain should be used!
func OptTestMain(m *testing.M) Opt {
	return OptFuncRunner(func() error {
		if code := m.Run(); code != 0 {
			return errors.New("tests have failed")
		}
		return nil
	})
}

// OptTestFunc allows wrapping testing.T into IntegrationTestRunner.
// Example TestApp:
//
//	func TestApp(t *testing.T) {
//		itr := it.NewIntegrationTestRunner(
//			it.OptBase("../"),
//			it.OptTarget("./cmd/app"),
//			it.OptCompose("docker-compose.yaml"),
//			it.OptWaitHTTPReady("http://127.0.0.1:8080", time.Second*10),
//			it.OptTestFunc(t, testApp),
//		)
//		if err := itr.InitAndRun(); err != nil {
//			t.Fatal(err)
//		}
//	}
//
//	func testApp(t *testing.T) {
//		// run tests here
//	}
//
// This pattern allows setting the env for each test separately.
func OptTestFunc(t *testing.T, fn func(*testing.T)) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.testRunner = func() error {
			fn(t)
			return nil
		}
		return nil
	}
}

// OptFuncRunner allows wrapping any function as testRunner function.
// This can be useful injecting code between ready and test code.
// Example TestMain:
//
//	func TestMain(m *testing.M) {
//		itr := it.NewIntegrationTestRunner(
//			it.OptBase("../"),
//			it.OptTarget("./cmd/app"),
//			it.OptCompose("docker-compose.yaml"),
//			it.OptWaitHTTPReady("http://127.0.0.1:8080", time.Second*10),
//			it.OptFuncRunner(func() error {
//				if err := initStuff(); err != nil {
//					return err
//				}
//				if code := m.Run(); code != 0 {
//					return errors.New("tests have failed")
//				}
//				return nil
//			}),
//		)
//		if err := itr.InitAndRun(); err != nil {
//			log.Fatal(err)
//		}
//	}
//
// Before using this pattern be sure to read how TestMain should be used!
func OptFuncRunner(fn func() error) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.testRunner = fn
		return nil
	}
}

// OptPreHandler adds handler as pre condition for tests to run.
func OptPreHandler(h PreHandler) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.preHandlers = append(itr.preHandlers, h)
		return nil
	}
}

// OptPreHandlerFn wraps functions as PreHandler and set's it as a pre condition for tests to run.
func OptPreHandlerFn(run, stop func() error) Opt {
	if run == nil {
		run = func() error { return nil }
	}
	if stop == nil {
		stop = func() error { return nil }
	}
	return OptPreHandler(&PreHandlerFn{RunFn: run, StopFn: stop})
}

// OptCompose adds docker compose stack as pre condition for tests to run.
func OptCompose(composeFile string, opts ...ComposeOpt) Opt {
	return func(itr *IntegrationTestRunner) error {
		compose, err := tc.NewDockerCompose(path.Join(itr.base, composeFile))
		if err != nil {
			return fmt.Errorf("failed to create new compose stack: %w", err)
		}

		c := &composeHandler{c: compose}
		for _, opt := range opts {
			opt(c)
		}

		itr.preHandlers = append(itr.preHandlers, c)
		return nil
	}
}

// OptWaitHTTPReady expects 200 OK from given url before tests can be started.
func OptWaitHTTPReady(url string, timeout time.Duration) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.ready = func() error {
			started := time.Now()
			for !isReady(url) {
				if time.Since(started) > timeout {
					return fmt.Errorf("readiness deadline %s exceeded", timeout)
				}
				time.Sleep(time.Millisecond * 100)
			}
			return nil
		}
		return nil
	}
}

func isReady(url string) bool {
	r, err := http.Get(url) //nolint:gosec
	if err != nil {
		return false
	}
	defer r.Body.Close()

	return r.StatusCode == http.StatusOK
}
