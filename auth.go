package common

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

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
	rt := newOauth2RoundTripper(conf, http.DefaultTransport)
	return &http.Client{Transport: rt}
}

type oauth2RoundTripper struct {
	config *ClientConfiguration
	rt     http.RoundTripper
	secret string
	mtx    sync.RWMutex
	client *http.Client
}

func newOauth2RoundTripper(config *ClientConfiguration, next http.RoundTripper) http.RoundTripper {
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
		// Either an inline secret or nothing (use an empty string) was provided.
		secret = string(r.config.ClientSecret)
	}

	if changed || r.rt == nil {
		config := &clientcredentials.Config{
			ClientID:       r.config.ClientID,
			ClientSecret:   secret,
			Scopes:         r.config.Scopes,
			TokenURL:       r.config.TokenURL,
			EndpointParams: r.config.EndpointParams,
		}

		var t http.RoundTripper

		client := &http.Client{Transport: t}
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
