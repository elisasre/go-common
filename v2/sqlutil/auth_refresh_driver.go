package sqlutil

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"

	"github.com/lib/pq"
)

// AuthProvider is used for refreshing DB credentials.
type AuthProvider interface {
	DSN() (string, error)
	IsAuthErr(error) bool
}

// AuthRefreshDriver wraps a sql.Driver with automatic credentials reloading.
type AuthRefreshDriver struct {
	driver    driver.Driver
	auth      AuthProvider
	latestDSN string
	lock      sync.Mutex
}

// NewAuthRefreshDriver wraps given sql.Driver and uses AuthLoader to fetch new DNS in case of auth error.
func NewAuthRefreshDriver(d driver.Driver, a AuthProvider) driver.Driver {
	return &AuthRefreshDriver{
		driver: d,
		auth:   a,
	}
}

// Open tries opening new connection and automatically refreshes credentials on Auth error.
func (d *AuthRefreshDriver) Open(_ string) (conn driver.Conn, err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	dsn := d.latestDSN
	if dsn == "" {
		if dsn, err = d.refreshDSN(); err != nil {
			return nil, err
		}
	}

	conn, err = d.driver.Open(dsn)
	if d.auth.IsAuthErr(err) {
		if dsn, err = d.refreshDSN(); err != nil {
			return nil, err
		}
		conn, err = d.driver.Open(dsn)
	}

	return conn, err
}

func (d *AuthRefreshDriver) refreshDSN() (dsn string, err error) {
	d.latestDSN, err = d.auth.DSN()
	return d.latestDSN, err
}

// OpenConnector return pointer to driver itself which implements also driver.Connector.
func (d *AuthRefreshDriver) OpenConnector(_ string) (driver.Connector, error) { return d, nil }

// Connect call's driver.Open which automatically refreshes credentials on Auth error.
func (d *AuthRefreshDriver) Connect(ctx context.Context) (driver.Conn, error) { return d.Open("") }

// Driver return pointer to itself.
func (d *AuthRefreshDriver) Driver() driver.Driver { return d }

// IsAuthenticationError checks if given error is know auth error.
func IsAuthenticationError(err error) bool {
	var pqError *pq.Error
	if errors.As(err, &pqError) {
		if pqError.Code.Class() == "28" {
			return true
		}
	}
	return false
}

// quoteDSNValue quotes a value for use in a PostgreSQL key=value DSN string.
// Values containing spaces, single quotes, or backslashes are wrapped in
// single quotes with internal single quotes and backslashes escaped.
func quoteDSNValue(v string) string {
	if v == "" {
		return "''"
	}
	if !strings.ContainsAny(v, ` \'`) {
		return v
	}
	replacer := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + replacer.Replace(v) + "'"
}
