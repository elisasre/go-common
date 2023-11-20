package sql

import (
	"context"
	"database/sql/driver"
	"sync"
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
