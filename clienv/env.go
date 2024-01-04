// Package clienv supports adding env variables automatically into github.com/urfave/cli flags.
package clienv

import (
	"strings"

	"github.com/urfave/cli/v2"
)

// AddEnvVars creates EnvVars slice for each flag in app.
func AddEnvVars(app *cli.App, prefix string) {
	for _, flag := range app.Flags {
		AddEnvVar(flag, prefix)
	}
}

// AddEnvVar sets f.EnvVar given that flag is has type.
func AddEnvVar(flag cli.Flag, prefix string) {
	envNames := NamesToEnv(prefix, flag.Names())
	switch f := flag.(type) {
	case *cli.BoolFlag:
		f.EnvVars = envNames
	case *cli.DurationFlag:
		f.EnvVars = envNames
	case *cli.Float64Flag:
		f.EnvVars = envNames
	case *cli.IntFlag:
		f.EnvVars = envNames
	case *cli.Int64Flag:
		f.EnvVars = envNames
	case *cli.IntSliceFlag:
		f.EnvVars = envNames
	case *cli.Int64SliceFlag:
		f.EnvVars = envNames
	case *cli.PathFlag:
		f.EnvVars = envNames
	case *cli.StringFlag:
		f.EnvVars = envNames
	case *cli.StringSliceFlag:
		f.EnvVars = envNames
	case *cli.TimestampFlag:
		f.EnvVars = envNames
	case *cli.UintFlag:
		f.EnvVars = envNames
	case *cli.Uint64Flag:
		f.EnvVars = envNames
	case *cli.UintSliceFlag:
		f.EnvVars = envNames
	case *cli.Uint64SliceFlag:
		f.EnvVars = envNames
	}
}

// NameToEnv creates new slice with matching env style names.
func NamesToEnv(prefix string, names []string) []string {
	envVars := make([]string, 0, len(names))
	for _, name := range names {
		envVars = append(envVars, NameToEnv(prefix, name))
	}
	return envVars
}

// NameToEnv converts names like foo-bar to FOO_BAR.
func NameToEnv(prefix, name string) string {
	envName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	if prefix == "" {
		return envName
	}
	return prefix + "_" + envName
}
