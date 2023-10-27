package log_test

import (
	"log/slog"

	"github.com/elisasre/go-common/log"
)

func ExampleNewDefaultEnvLogger() {
	log.NewDefaultEnvLogger()
	slog.Info("Hello world")
	slog.Error("Some error")
}
