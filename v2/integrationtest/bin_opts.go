package integrationtest

import (
	"fmt"
	"io"
	"path/filepath"
)

// BinOpt is option type for BinHandler.
type BinOpt func(*BinHandler) error

// BinOptTarget sets path to compilation target.
func BinOptTarget(target string) BinOpt {
	return func(bh *BinHandler) error {
		bh.target = target
		return nil
	}
}

// BinOptBase sets execution base path BinHandler.
// BinOptBase should be usually the first option when passing options to NewIntegrationTestRunner.
func BinOptBase(base string) BinOpt {
	return func(bh *BinHandler) error {
		absBase, err := filepath.Abs(base)
		if err != nil {
			return fmt.Errorf("getting absolute path for base: '%s' failed: %w", base, err)
		}

		bh.base = absBase
		return nil
	}
}

// BinOptOutput sets output for compilation target.
func BinOptOutput(output string) BinOpt {
	return func(bh *BinHandler) error {
		bh.bin = output
		return nil
	}
}

// BinOptRunArgs adds args to run arguments for test binary.
func BinOptRunArgs(args ...string) BinOpt {
	return func(bh *BinHandler) error {
		bh.runArgs = append(bh.runArgs, args...)
		return nil
	}
}

// BinOptSetRunArgs sets run arguments for test binary.
func BinOptSetRunArgs(args ...string) BinOpt {
	return func(bh *BinHandler) error {
		bh.runArgs = args
		return nil
	}
}

// BinOptBuildArgs adds args to build arguments for test binary.
func BinOptBuildArgs(args ...string) BinOpt {
	return func(bh *BinHandler) error {
		bh.buildArgs = append(bh.buildArgs, args...)
		return nil
	}
}

// BinOptRunEnv adds env to test binary's run env.
func BinOptRunEnv(env ...string) BinOpt {
	return func(bh *BinHandler) error {
		bh.runEnv = append(bh.runEnv, env...)
		return nil
	}
}

// BinOptSetRunEnv sets test binary's run env.
func BinOptSetRunEnv(env ...string) BinOpt {
	return func(bh *BinHandler) error {
		bh.runEnv = env
		return nil
	}
}

// BinOptBuildEnv adds env to test binary's build env.
func BinOptBuildEnv(env ...string) BinOpt {
	return func(bh *BinHandler) error {
		bh.buildEnv = append(bh.buildEnv, env...)
		return nil
	}
}

// BinOptCoverDir sets coverage directory for test binary.
func BinOptCoverDir(coverDir string) BinOpt {
	return func(bh *BinHandler) error {
		bh.coverDir = coverDir
		return nil
	}
}

// BinOptRunStdout sets stdout for test binary.
func BinOptRunStdout(stdout io.Writer) BinOpt {
	return func(bh *BinHandler) error {
		bh.runStdout = stdout
		return nil
	}
}

// BinOptRunStderr sets stderr for test binary.
func BinOptRunStderr(stderr io.Writer) BinOpt {
	return func(bh *BinHandler) error {
		bh.runStderr = stderr
		return nil
	}
}
