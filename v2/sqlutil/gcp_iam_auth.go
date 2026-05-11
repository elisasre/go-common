package sqlutil

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const gcpCloudSQLLoginScope = "https://www.googleapis.com/auth/sqlservice.login"

// TokenSourceFn creates an oauth2.TokenSource for authenticating to Cloud SQL.
// The default implementation uses google.FindDefaultCredentials with the Cloud SQL login scope.
type TokenSourceFn func(ctx context.Context) (oauth2.TokenSource, error)

// GCPIAMAuth implements AuthProvider for Google Cloud SQL IAM database authentication.
// It generates OAuth2 access tokens and uses them as the PostgreSQL password.
// This approach requires direct network access to the Cloud SQL instance
// (Private IP / VPC, or authorized networks for Public IP).
type GCPIAMAuth struct {
	Host    string
	Port    int
	DBUser  string
	DBName  string
	SSLMode string

	password      string
	tokenSourceFn TokenSourceFn
}

// GCPIAMOption configures a GCPIAMAuth instance.
type GCPIAMOption func(*GCPIAMAuth)

// WithGCPTokenSource overrides the default credential source with a custom oauth2.TokenSource.
// This is useful for testing or when using explicit service account credentials.
func WithGCPTokenSource(ts oauth2.TokenSource) GCPIAMOption {
	return func(g *GCPIAMAuth) {
		g.tokenSourceFn = func(_ context.Context) (oauth2.TokenSource, error) {
			return ts, nil
		}
	}
}

// WithSSLMode overrides the default SSL mode ("require") for the PostgreSQL connection.
func WithSSLMode(mode string) GCPIAMOption {
	return func(g *GCPIAMAuth) {
		g.SSLMode = mode
	}
}

// WithGCPTokenSourceFn overrides the function used to create the oauth2.TokenSource.
// Unlike WithGCPTokenSource, this allows lazy initialization and context propagation.
func WithGCPTokenSourceFn(fn TokenSourceFn) GCPIAMOption {
	return func(g *GCPIAMAuth) {
		g.tokenSourceFn = fn
	}
}

// NewGCPIAMAuth creates a new GCPIAMAuth that uses OAuth2 access tokens as the
// PostgreSQL password for Cloud SQL IAM authentication.
//
// Use with AuthRefreshDriver to get automatic credential refresh on auth errors:
//
//	auth := sqlutil.NewGCPIAMAuth("10.0.0.1", 5432, "sa@project.iam", "mydb")
//	driver := sqlutil.NewAuthRefreshDriver(&pq.Driver{}, auth)
//	sql.Register("cloudsql-iam", driver)
func NewGCPIAMAuth(host string, port int, dbUser, dbName string, opts ...GCPIAMOption) *GCPIAMAuth {
	g := &GCPIAMAuth{
		Host:    host,
		Port:    port,
		DBUser:  dbUser,
		DBName:  dbName,
		SSLMode: "require",
		tokenSourceFn: func(ctx context.Context) (oauth2.TokenSource, error) {
			creds, err := google.FindDefaultCredentials(ctx, gcpCloudSQLLoginScope)
			if err != nil {
				return nil, fmt.Errorf("find GCP credentials: %w", err)
			}
			return creds.TokenSource, nil
		},
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// RefreshPassword obtains a fresh OAuth2 access token and stores it as the password.
func (g *GCPIAMAuth) RefreshPassword() error {
	ctx := context.Background()

	ts, err := g.tokenSourceFn(ctx)
	if err != nil {
		return fmt.Errorf("create token source: %w", err)
	}

	token, err := ts.Token()
	if err != nil {
		return fmt.Errorf("obtain access token: %w", err)
	}

	g.password = token.AccessToken
	return nil
}

// DSN refreshes the OAuth2 token and returns a PostgreSQL connection string.
func (g *GCPIAMAuth) DSN() (string, error) {
	if err := g.RefreshPassword(); err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=10",
		g.Host, g.Port, g.DBUser, quoteDSNValue(g.password), g.DBName, g.SSLMode,
	), nil
}

// IsAuthErr checks if the given error is a PostgreSQL authentication error.
func (*GCPIAMAuth) IsAuthErr(err error) bool {
	return IsAuthenticationError(err)
}

// Compile-time check that GCPIAMAuth implements AuthProvider.
var _ AuthProvider = (*GCPIAMAuth)(nil)
