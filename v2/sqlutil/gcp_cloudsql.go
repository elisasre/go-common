package sqlutil

import (
	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/postgres/pgxv5"
)

// CloudSQLDriver registers a database/sql driver backed by the Cloud SQL Go Connector.
// The connector handles TLS, certificate rotation, and IAM token refresh automatically.
//
// The returned cleanup function must be called when the driver is no longer needed
// (typically deferred in main).
//
// DSN format uses the Cloud SQL instance connection name as the host:
//
//	"host=project:region:instance user=sa@project.iam dbname=mydb sslmode=disable"
//
// Example usage:
//
//	cleanup, err := sqlutil.CloudSQLDriver("cloudsql-postgres",
//	    cloudsqlconn.WithIAMAuthN(),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cleanup()
//
//	db, err := sql.Open("cloudsql-postgres",
//	    "host=project:region:instance user=sa@project.iam dbname=mydb sslmode=disable",
//	)
func CloudSQLDriver(name string, opts ...cloudsqlconn.Option) (cleanup func() error, err error) {
	return pgxv5.RegisterDriver(name, opts...)
}
