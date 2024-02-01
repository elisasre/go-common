package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/elisasre/go-common"
	database "github.com/elisasre/go-common/auth/db"
	"github.com/elisasre/go-common/auth/memory"
)

// AuthInterface will contain interface to interact with different auth providers.
type AuthInterface interface {
	GetKeys() []common.JWTKey
	GetCurrentKey() common.JWTKey
	RotateKeys(context.Context) error
	RefreshKeys(context.Context, bool) ([]common.JWTKey, error)
}

func AuthProvider(ctx context.Context, mode string, store common.Datastore) (AuthInterface, error) {
	slog.Info("Using AuthProvider",
		slog.String("mode", mode))
	switch mode {
	case "memory":
		return memory.NewMemory(ctx)
	case "database":
		return database.NewDatabase(ctx, store)
	default:
		return nil, fmt.Errorf("unknown auth mode '%s'", mode)
	}
}
