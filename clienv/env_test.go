// Package clienv supports adding env variables automatically into github.com/urfave/cli flags.
package clienv_test

import (
	"testing"
	"time"

	"github.com/elisasre/go-common/clienv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestAddEnvVars(t *testing.T) {
	testTime, err := time.Parse(time.Kitchen, "1:47PM")
	require.NoError(t, err)

	t.Setenv("TEST_BOOL_FLAG", "true")
	t.Setenv("TEST_DURATION_FLAG", "1m")
	t.Setenv("TEST_FLOAT64_FLAG", "1.234")
	t.Setenv("TEST_INT_FLAG", "-5")
	t.Setenv("TEST_INT64_FLAG", "-9000")
	t.Setenv("TEST_INT_SLICE_FLAG", "-1,2,-3,4")
	t.Setenv("TEST_INT64_SLICE_FLAG", "5,-6,7,-8")
	t.Setenv("TEST_PATH_FLAG", "./some/other/path")
	t.Setenv("TEST_STRING_FLAG", "test")
	t.Setenv("TEST_STRING_SLICE_FLAG", "a,b,c")
	t.Setenv("TEST_TIMESTAMP_FLAG", "1:47PM")
	t.Setenv("TEST_UINT_FLAG", "2")
	t.Setenv("TEST_UINT64_FLAG", "2000")
	t.Setenv("TEST_UINT_SLICE_FLAG", "3,7,9")
	t.Setenv("TEST_UINT64_SLICE_FLAG", "2342342,56,7")

	app := &cli.App{
		Name: "test",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "bool-flag"},
			&cli.DurationFlag{Name: "duration-flag"},
			&cli.Float64Flag{Name: "float64-flag"},
			&cli.IntFlag{Name: "int-flag"},
			&cli.Int64Flag{Name: "int64-flag"},
			&cli.IntSliceFlag{Name: "int-slice-flag"},
			&cli.Int64SliceFlag{Name: "int64-slice-flag"},
			&cli.PathFlag{Name: "path-flag"},
			&cli.StringFlag{Name: "string-flag"},
			&cli.StringSliceFlag{Name: "string-slice-flag"},
			&cli.TimestampFlag{Name: "timestamp-flag", Layout: time.Kitchen},
			&cli.UintFlag{Name: "uint-flag"},
			&cli.Uint64Flag{Name: "uint64-flag"},
			&cli.UintSliceFlag{Name: "uint-slice-flag"},
			&cli.Uint64SliceFlag{Name: "uint64-slice-flag"},
		},
		Action: func(ctx *cli.Context) error {
			assert.Equal(t, true, ctx.Bool("bool-flag"))
			assert.Equal(t, time.Minute, ctx.Duration("duration-flag"))
			assert.Equal(t, 1.234, ctx.Float64("float64-flag"))
			assert.Equal(t, int(-5), ctx.Int("int-flag"))
			assert.Equal(t, int64(-9000), ctx.Int64("int64-flag"))
			assert.Equal(t, []int{-1, 2, -3, 4}, ctx.IntSlice("int-slice-flag"))
			assert.Equal(t, []int64{5, -6, 7, -8}, ctx.Int64Slice("int64-slice-flag"))
			assert.Equal(t, "./some/other/path", ctx.Path("path-flag"))
			assert.Equal(t, "test", ctx.String("string-flag"))
			assert.Equal(t, []string{"a", "b", "c"}, ctx.StringSlice("string-slice-flag"))
			assert.Equal(t, testTime, *ctx.Timestamp("timestamp-flag"))
			assert.Equal(t, uint(2), ctx.Uint("uint-flag"))
			assert.Equal(t, uint64(2000), ctx.Uint64("uint64-flag"))
			assert.Equal(t, []uint{3, 7, 9}, ctx.UintSlice("uint-slice-flag"))
			assert.Equal(t, []uint64{2342342, 56, 7}, ctx.Uint64Slice("uint64-slice-flag"))
			return nil
		},
	}
	clienv.AddEnvVars(app, "TEST")
	err = app.Run(nil)
	require.NoError(t, err)
}

func TestNameToEnv(t *testing.T) {
	tcs := []struct {
		name     string
		prefix   string
		flagName string
		expected string
	}{
		{
			name:     "env var without prefix",
			prefix:   "",
			flagName: "flag-no-prefix",
			expected: "FLAG_NO_PREFIX",
		},
		{
			name:     "env var with prefix",
			prefix:   "prefix",
			flagName: "flag-with-prefix",
			expected: "PREFIX_FLAG_WITH_PREFIX",
		},
		{
			name:     "lowercase prefix gets converted to uppercase",
			prefix:   "prefix",
			flagName: "other-flag-with-prefix",
			expected: "PREFIX_OTHER_FLAG_WITH_PREFIX",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := clienv.NameToEnv(tc.prefix, tc.flagName)
			assert.Equal(t, tc.expected, output)
		})
	}
}
