package httpserver_test

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/elisasre/go-common/service/module/httpserver"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrOpt(t *testing.T) {
	optErr := errors.New("opt err")
	srv := httpserver.New(func(s *httpserver.Server) error { return optErr })
	err := srv.Init()
	require.ErrorIs(t, err, optErr)
}

func TestListenErr(t *testing.T) {
	srv := httpserver.New(httpserver.WithAddr(`sdf./43/s]\\][]"`))
	err := srv.Init()
	require.Error(t, err)
}

func TestHTTPS(t *testing.T) {
	srv := httpserver.New(httpserver.WithServer(&http.Server{
		Addr:              "127.0.0.1:0",
		ReadHeaderTimeout: time.Second,
		TLSConfig:         &tls.Config{}, //nolint:gosec
	}))
	err := srv.Init()
	require.NoError(t, err)
	require.Contains(t, srv.URL(), "https://127.0.0.1:")
}

func TestServer(t *testing.T) {
	srv := httpserver.New(
		httpserver.WithServer(&http.Server{ReadHeaderTimeout: time.Second}),
		httpserver.WithAddr("127.0.0.1:0"),
		httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Hello")
		})),
	)

	require.NotEmpty(t, srv.Name())
	require.NoError(t, srv.Init())
	wg := &multierror.Group{}
	wg.Go(srv.Run)

	assertGet(t, srv.URL(), "Hello")

	assert.NoError(t, srv.Stop())
	err := wg.Wait().ErrorOrNil()
	require.NoError(t, err)
}

func TestServerShutdownTimeout(t *testing.T) {
	block := make(chan struct{})
	srv := httpserver.New(
		httpserver.WithAddr("127.0.0.1:0"),
		httpserver.WithShutdownTimeout(time.Second),
		httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-block
		})),
	)
	err := srv.Init()
	require.NoError(t, err)
	url := srv.URL()

	wg := &multierror.Group{}
	wg.Go(srv.Run)

	go func() {
		resp, err := http.Get(url) //nolint:gosec
		assert.NoError(t, err)
		io.Copy(io.Discard, resp.Body) //nolint: errcheck
	}()

	time.Sleep(time.Millisecond * 10)
	err = srv.Stop()
	assert.ErrorContains(t, err, "context deadline exceeded")

	err = wg.Wait().ErrorOrNil()
	assert.NoError(t, err)
}

func assertGet(t testing.TB, url, body string) {
	resp, err := http.Get(url) //nolint:gosec
	if !assert.NoError(t, err) {
		return
	}

	data, err := io.ReadAll(resp.Body)
	if !assert.NoError(t, err) {
		return
	}

	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, body, string(data))
}
