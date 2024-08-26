package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/elisasre/go-common/v2/auth/cache/cachetest"
	"github.com/elisasre/go-common/v2/auth/store/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrestc "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRotateKeys(t *testing.T) {
	dbName := "postgres"
	postgresContainer, err := postgrestc.Run(context.Background(),
		"postgres:16",
		postgrestc.WithDatabase(dbName),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)

	dsn, err := postgresContainer.ConnectionString(context.Background(), "sslmode=disable")
	require.NoError(t, err)

	db, err := sqlx.Open("postgres", dsn)
	require.NoError(t, err)

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS jwt_keys (
		id BIGSERIAL PRIMARY KEY,
		created_at timestamp with time zone,
		updated_at timestamp with time zone,
		deleted_at timestamp with time zone,
		k_id text,
		private_key_as_bytes bytea,
		public_key_as_bytes bytea
	);`)
	require.NoError(t, err)

	store, err := postgres.New(
		postgres.WithSqlxDB(db),
		postgres.WithSecret("secret"),
	)
	require.NoError(t, err)

	cachetest.RunSuite(t, store)
}
