package sqlutil

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
)

type BuildAuthTokenFn func(ctx context.Context,
	endpoint, region, dbUser string,
	creds aws.CredentialsProvider,
	optFns ...func(options *auth.BuildAuthTokenOptions)) (string, error)

// IAMAuth implements AuthProvider and generates IAM DB credentials.
type IAMAuth struct {
	Host, DBUser, DBPassword, DBRegion, SSLMode, DBName string
	Port                                                int
	BuildAuthToken                                      BuildAuthTokenFn
}

func (i *IAMAuth) RefreshPassword() error {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s:%d", i.Host, i.Port)
	authToken, err := i.BuildAuthToken(ctx, endpoint, i.DBRegion, i.DBUser, cfg.Credentials)
	if err != nil {
		return err
	}

	i.DBPassword = authToken

	return nil
}

// DSN will trigger reparsing of configuration and return DSN or an error.
func (i *IAMAuth) DSN() (string, error) {
	if err := i.RefreshPassword(); err != nil {
		return "", err
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=10",
		i.Host, i.Port, i.DBUser, quoteDSNValue(i.DBPassword), i.DBName, i.SSLMode), nil
}

// IsAuthErr checks if given error is know auth error.
func (*IAMAuth) IsAuthErr(err error) bool {
	return IsAuthenticationError(err)
}
