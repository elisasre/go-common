package httpserver_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/elisasre/go-common/v2/service/module/httpserver"
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

func TestHTTPSWithRequest(t *testing.T) {
	tlsConfig := generateTLSConfig(t)

	srv := httpserver.New(
		httpserver.WithAddr("127.0.0.1:0"),
		httpserver.WithTLSConfig(tlsConfig),
		httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Hello HTTPS")
		})),
	)

	require.NoError(t, srv.Init())
	require.Contains(t, srv.URL(), "https://127.0.0.1:")

	wg := &multierror.Group{}
	wg.Go(srv.Run)

	// Create HTTP client that skips certificate verification for testing
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
	resp, err := client.Get(srv.URL())
	require.NoError(t, err)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, "Hello HTTPS", string(data))

	assert.NoError(t, srv.Stop())
	err = wg.Wait().ErrorOrNil()
	require.NoError(t, err)
}

// generateTLSConfig creates a self-signed certificate for testing.
func generateTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Co"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Encode private key to PEM
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes})

	// Create TLS certificate
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
}

func TestServer(t *testing.T) {
	srv := httpserver.New(
		httpserver.WithServer(&http.Server{ReadHeaderTimeout: time.Second}),
		httpserver.WithAddr("127.0.0.1:0"),
		httpserver.WithName("test-server"),
		httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Hello")
		})),
	)

	require.NotEmpty(t, srv.Name())
	require.NoError(t, srv.Init())
	wg := &multierror.Group{}
	wg.Go(srv.Run)

	assertGet(t, srv.URL(), "Hello")
	assert.Equal(t, "test-server", srv.Name())

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
