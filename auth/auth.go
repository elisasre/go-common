package auth

import (
	"context"
	"fmt"

	"github.com/elisasre/go-common"
	database "github.com/elisasre/go-common/auth/db"
	"github.com/elisasre/go-common/auth/memory"

	"github.com/rs/zerolog/log"
)

// AuthInterface will contain interface to interact with different auth providers.
type AuthInterface interface {
	GetKeys() []common.JWTKey
	GetCurrentKey() common.JWTKey
	RotateKeys(context.Context) error
	RefreshKeys(context.Context, bool) ([]common.JWTKey, error)
}

func AuthProvider(ctx context.Context, mode string, store common.Datastore) (AuthInterface, error) {
	log.Info().Str("mode", mode).Msg("Using AuthProvider")
	switch mode {
	case "memory":
		return memory.NewMemory()
	case "database":
		return database.NewDatabase(ctx, store)
	default:
		return nil, fmt.Errorf("unknown auth mode '%s'", mode)
	}
}
