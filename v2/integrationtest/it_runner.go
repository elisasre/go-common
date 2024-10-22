// Package integrationtest makes it easier to run integration tests against compiled binary.
package integrationtest

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
)

// Default file locations.
const (
	BinDir                  = "target/testbin/"
	IntegrationTestCoverDir = "./target/tests/cover/int/"
)

type IntegrationTestRunner struct {
	base        string
	preHandlers []PreHandler
	binHandler  *BinHandler
	ready       func() error
	testRunner  func() error
	opts        []Opt
}

// NewIntegrationTestRunner creates new IntegrationTestRunner with given options.
func NewIntegrationTestRunner(opts ...Opt) *IntegrationTestRunner {
	return &IntegrationTestRunner{
		opts:       opts,
		ready:      func() error { return nil },
		base:       ".",
		binHandler: NewBinHandler(),
	}
}

// InitAndRun combines Init() and Run() calls for convenience.
func (itr *IntegrationTestRunner) InitAndRun() error {
	if err := itr.Init(); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	if err := itr.Run(); err != nil {
		return fmt.Errorf("run failed: %w", err)
	}

	return nil
}

// Init applies all options to test runner.
func (itr *IntegrationTestRunner) Init() error {
	for _, opt := range itr.opts {
		if err := opt(itr); err != nil {
			return err
		}
	}

	// If no base path was provided let's apply proper checks for default path.
	if itr.base == "." {
		if err := OptBase(itr.base)(itr); err != nil {
			return err
		}
	}

	return nil
}

// Run starts test workflow.
//
// Workflow steps:
// 1. Run all preSetup steps
// 2. Build test binary
// 3. Start test binary in background
// 4. Execute test function
// 5. Cleanup resources
//
// The contents of above steps will depend on given options.
func (itr *IntegrationTestRunner) Run() error {
	if err := itr.preSetup(); err != nil {
		return err
	}

	if err := itr.buildAndRunBin(); err != nil {
		_ = itr.cleanup()
		return err
	}

	if err := itr.testRunner(); err != nil {
		_ = itr.cleanup()
		return fmt.Errorf("running integration tests failed: %w", err)
	}

	return itr.cleanup()
}

func (itr *IntegrationTestRunner) preSetup() error {
	for _, h := range itr.preHandlers {
		if err := h.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (itr *IntegrationTestRunner) buildAndRunBin() error {
	if err := itr.binHandler.Build(); err != nil {
		return fmt.Errorf("building applicfunc Teation failed: %w", err)
	}

	if err := itr.binHandler.Start(); err != nil {
		return fmt.Errorf("starting application failed: %w", err)
	}

	return itr.ready()
}

func (itr *IntegrationTestRunner) cleanup() error {
	var result error
	if err := itr.binHandler.Stop(); err != nil {
		result = multierror.Append(result, fmt.Errorf("running application failed: %w", err))
	}

	for i := len(itr.preHandlers) - 1; i >= 0; i-- {
		if err := itr.preHandlers[i].Stop(); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}

// PreHandler is used for setting up pre conditions for test.
// Run() allows to perform setup before tests and Stop() the cleanup after the tests.
type PreHandler interface {
	Run() error
	Stop() error
}

type PreHandlerFn struct {
	RunFn, StopFn func() error
}

func (h *PreHandlerFn) Run() error  { return h.RunFn() }
func (h *PreHandlerFn) Stop() error { return h.StopFn() }
