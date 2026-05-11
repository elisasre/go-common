package sqlutil_test

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/elisasre/go-common/v2/sqlutil"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrestc "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/oauth2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGCPIAMAuth(t *testing.T) {
	ctx := context.Background()
	dbTestUserName := "gcptestuser"
	dbTestUserOriginalPassword := "original-token"
	dbTestUserNewPassword := "refreshed-token"
	dbName := "postgres"

	postgresContainer, err := postgrestc.Run(ctx,
		"postgres:16",
		postgrestc.WithDatabase(dbName),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)

	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	createUserQuery := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s';", dbTestUserName, dbTestUserOriginalPassword)
	err = adminDB.Exec(createUserQuery).Error
	require.NoError(t, err)

	dbEndpoint, err := postgresContainer.Endpoint(ctx, "tcp")
	require.NoError(t, err)
	dbHost, dbPortStr, err := net.SplitHostPort(strings.TrimPrefix(dbEndpoint, "tcp://"))
	require.NoError(t, err)
	dbPort, err := strconv.Atoi(dbPortStr)
	require.NoError(t, err)

	// Mock token source that returns passwords from a pre-defined list,
	// simulating OAuth2 token refresh behavior.
	passwords := []string{dbTestUserOriginalPassword, dbTestUserNewPassword}
	currentIndex := 0
	mockTokenSource := tokenFunc(func() (*oauth2.Token, error) {
		if currentIndex >= len(passwords) {
			return nil, fmt.Errorf("no more mock tokens available")
		}
		token := &oauth2.Token{
			AccessToken: passwords[currentIndex],
			TokenType:   "Bearer",
		}
		currentIndex++
		return token, nil
	})

	auth := sqlutil.NewGCPIAMAuth(
		dbHost, dbPort, dbTestUserName, dbName,
		sqlutil.WithGCPTokenSource(mockTokenSource),
		sqlutil.WithSSLMode("disable"),
	)

	driverName := fmt.Sprintf("postgres-gcp-iam-%d", time.Now().UnixNano())
	driver := sqlutil.NewAuthRefreshDriver(&pq.Driver{}, auth)
	sql.Register(driverName, driver)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DriverName: driverName,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	require.NoError(t, err, dbEndpoint)

	// Ping should succeed with original password.
	testDB, err := db.DB()
	require.NoError(t, err, "failed to get db connection")
	err = testDB.Ping()
	require.NoError(t, err, "failed to ping db")

	// Change password and terminate all connections for the test user.
	changePasswordQuery := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s';", dbTestUserName, dbTestUserNewPassword)
	err = adminDB.Exec(changePasswordQuery).Error
	require.NoError(t, err, "failed to change password")

	terminateConnectionQuery := fmt.Sprintf(
		//nolint:lll
		"SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '%s' AND pg_stat_activity.usename = '%s' AND pid <> pg_backend_pid();",
		dbName, dbTestUserName)
	err = adminDB.Exec(terminateConnectionQuery).Error
	require.NoError(t, err, "failed to terminate connections")

	// Ping should fail with stale connection.
	err = testDB.Ping()
	require.Error(t, err, "ping should fail after password change")

	// Get new connection — AuthRefreshDriver should auto-refresh the token.
	testDB, err = db.DB()
	require.NoError(t, err, "failed to get db connection")
	err = testDB.Ping()
	require.NoError(t, err, "failed to ping db after token refresh")
}

func TestGCPIAMAuth_IsAuthErr(t *testing.T) {
	auth := sqlutil.NewGCPIAMAuth("localhost", 5432, "user", "db")

	t.Run("pq auth error", func(t *testing.T) {
		err := &pq.Error{Code: "28P01"} // invalid_password
		require.True(t, auth.IsAuthErr(err))
	})

	t.Run("pq non-auth error", func(t *testing.T) {
		err := &pq.Error{Code: "42P01"} // undefined_table
		require.False(t, auth.IsAuthErr(err))
	})

	t.Run("non-pq error", func(t *testing.T) {
		err := fmt.Errorf("some other error")
		require.False(t, auth.IsAuthErr(err))
	})

	t.Run("nil error", func(t *testing.T) {
		require.False(t, auth.IsAuthErr(nil))
	})
}

// tokenFunc adapts a function to the oauth2.TokenSource interface.
type tokenFunc func() (*oauth2.Token, error)

func (f tokenFunc) Token() (*oauth2.Token, error) { return f() }
