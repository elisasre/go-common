package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" //nolint:gosec // G501: Blocklisted import crypto/md5: weak cryptographic primitive
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// JWTKey is struct for storing auth private keys.
type JWTKey struct {
	CreatedAt  time.Time       `yaml:"created_at" json:"created_at"`
	KID        string          `yaml:"kid" json:"kid"`
	PrivateKey *rsa.PrivateKey `yaml:"-" json:"-"`
	PublicKey  *rsa.PublicKey  `yaml:"-" json:"-"`
}
type OAuth2 struct {
	ClientID         string
	ClientSecret     string
	ClientSecretFile string
	Scopes           []string
	TokenURL         string
	EndpointParams   url.Values
}

type ClientConfiguration struct {
	OAuth2
}

func NewClient(ctx context.Context, conf *ClientConfiguration) *http.Client {
	rt := newOauth2RoundTripper(conf)
	return &http.Client{Transport: otelhttp.NewTransport(rt)}
}

type oauth2RoundTripper struct {
	config *ClientConfiguration
	rt     http.RoundTripper
	secret string
	mtx    sync.RWMutex
	client *http.Client
}

func newOauth2RoundTripper(config *ClientConfiguration) http.RoundTripper {
	return &oauth2RoundTripper{
		config: config,
	}
}

func (r *oauth2RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		secret  string
		changed bool
	)

	if r.config.ClientSecretFile != "" {
		data, err := os.ReadFile(r.config.ClientSecretFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read oauth2 client secret file %s: %w", r.config.ClientSecretFile, err)
		}
		secret = strings.TrimSpace(string(data))
		r.mtx.RLock()
		changed = secret != r.secret
		r.mtx.RUnlock()
	} else {
		secret = r.config.ClientSecret
	}

	if changed || r.rt == nil {
		config := &clientcredentials.Config{
			ClientID:       r.config.ClientID,
			ClientSecret:   secret,
			Scopes:         r.config.Scopes,
			TokenURL:       r.config.TokenURL,
			EndpointParams: r.config.EndpointParams,
		}

		client := &http.Client{}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
		tokenSource := config.TokenSource(ctx)

		r.mtx.Lock()
		r.secret = secret
		r.rt = &oauth2.Transport{
			Base:   nil,
			Source: tokenSource,
		}
		if r.client != nil {
			r.client.CloseIdleConnections()
		}
		r.client = client
		r.mtx.Unlock()
	}

	r.mtx.RLock()
	currentRT := r.rt
	r.mtx.RUnlock()
	return currentRT.RoundTrip(req)
}

// BasicAuth returns a base64 encoded string of the user and password.
func BasicAuth(user, password string) string {
	auth := fmt.Sprintf("%s:%s", user, password)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// Base64decode decodes base64 input to string.
func Base64decode(v string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	return string(data), nil
}

func createHash(key string) string {
	hasher := md5.New() //nolint:gosec // G501: Blocklisted import crypto/md5: weak cryptographic primitive
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// Encrypt the secret input with passphrase
// source https://www.thepolyglotdeveloper.com/2018/02/encrypt-decrypt-data-golang-application-crypto-packages/
func Encrypt(data []byte, passphrase string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(createHash(passphrase)))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt the encrypted secret with passphrase.
func Decrypt(data []byte, passphrase string) ([]byte, error) {
	key := []byte(createHash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("invalid data")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open([]byte{}, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// GenerateNewKeyPair generates new private and public keys.
func GenerateNewKeyPair() (JWTKey, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return JWTKey{}, fmt.Errorf("error generating RSA private key: %w", err)
	}

	err = rsaKey.Validate()
	if err != nil {
		return JWTKey{}, err
	}

	serial, err := BuildPKISerial()
	if err != nil {
		return JWTKey{}, err
	}

	return JWTKey{
		KID:        serial.String(),
		PrivateKey: rsaKey,
		PublicKey:  &rsaKey.PublicKey,
	}, nil
}

// BuildPKISerial generates random big.Int.
func BuildPKISerial() (*big.Int, error) {
	randomLimit := new(big.Int).Lsh(big.NewInt(1), 32)
	randomComponent, err := rand.Int(rand.Reader, randomLimit)
	if err != nil {
		return nil, fmt.Errorf("error generating random number: %w", err)
	}

	serial := big.NewInt(time.Now().UnixNano())
	serial.Lsh(serial, 32)
	serial.Or(serial, randomComponent)

	return serial, nil
}

func EncodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	}
	return pem.EncodeToMemory(&privBlock)
}

func EncodePublicKeyToPEM(publicKey *rsa.PublicKey) []byte {
	privBlock := pem.Block{
		Type:    "RSA PUBLIC KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PublicKey(publicKey),
	}
	return pem.EncodeToMemory(&privBlock)
}
