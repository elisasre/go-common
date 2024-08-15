package postgres_test

import (
	"testing"

	"github.com/elisasre/go-common/v2/auth/cache/cachetest"
	"github.com/elisasre/go-common/v2/auth/store/postgres"
	"github.com/stretchr/testify/require"
)

func TestRotateKeys(t *testing.T) {
	t.Skip("TODO: add postgres container")
	db, err := postgres.New()
	require.NoError(t, err)
	cachetest.RunSuite(t, db)
}
