package sql

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrestc "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestIAMAuth(t *testing.T) {
	ctx := context.Background()
	dbTestUserName := "testuser"
	dbTestUserOriginalPassword := "pencil"
	dbTestUserNewPassword := "hunter2"
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

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	createUserQuery := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s';", dbTestUserName, dbTestUserOriginalPassword)
	err = adminDB.Exec(createUserQuery).Error
	require.NoError(t, err)

	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	// Mock authentication token generator
	mockBuildAuthToken := func(passwords []string) func(ctx context.Context,
		endpoint, region, dbUser string,
		creds aws.CredentialsProvider,
		optFns ...func(options *auth.BuildAuthTokenOptions),
	) (string, error) {
		currentPasswordIndex := 0
		return func(ctx context.Context,
			endpoint, region, dbUser string,
			creds aws.CredentialsProvider,
			optFns ...func(options *auth.BuildAuthTokenOptions),
		) (string, error) {
			password := passwords[currentPasswordIndex]
			currentPasswordIndex++
			return password, nil
		}
	}

	dbEndpoint, err := postgresContainer.Endpoint(ctx, "tcp")
	require.NoError(t, err)
	dbHost, dbPortStr, err := net.SplitHostPort(strings.TrimPrefix(dbEndpoint, "tcp://"))
	require.NoError(t, err)
	dbPort, err := strconv.Atoi(dbPortStr)
	require.NoError(t, err)

	driverName := fmt.Sprintf("postgres-dyn-auth-%d", time.Now().UnixNano())
	driver := NewAuthRefreshDriver(&pq.Driver{}, &IAMAuth{
		Host:           dbHost,
		Port:           dbPort,
		DBUser:         dbTestUserName,
		DBRegion:       "mock",
		SSLMode:        "disable",
		DBName:         "postgres",
		BuildAuthToken: mockBuildAuthToken([]string{dbTestUserOriginalPassword, dbTestUserNewPassword}),
	})
	sql.Register(driverName, driver)
	db, err := gorm.Open(postgres.New(postgres.Config{
		DriverName:           driverName,
		PreferSimpleProtocol: false,
		WithoutReturning:     false,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	require.NoError(t, err, dbEndpoint)

	// Ping should succeed
	testDB, err := db.DB()
	require.NoError(t, err, "failed to get db connection")
	err = testDB.Ping()
	require.NoError(t, err, "failed to ping db")

	// Change password and terminate connection
	changePasswordQuery := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s';", dbTestUserName, dbTestUserNewPassword)
	err = adminDB.Exec(changePasswordQuery).Error
	require.NoError(t, err, "failed to change password")
	terminateConnectionQuery := fmt.Sprintf(
		//nolint:lll
		"SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '%s' AND pg_stat_activity.usename = '%s' AND pid <> pg_backend_pid();",
		dbName, dbTestUserName)
	err = adminDB.Exec(terminateConnectionQuery).Error
	require.NoError(t, err, "failed to terminate connections")

	// Ping should fail
	err = testDB.Ping()
	require.Error(t, err, "ping should fail after password change")

	// Get new connection, ping should succeed
	testDB, err = db.DB()
	require.NoError(t, err, "failed to get db connection")
	err = testDB.Ping()
	require.NoError(t, err, "failed to ping db")
}
