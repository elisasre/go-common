package common

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func NewClient(ctx context.Context, conf *clientcredentials.Config, secret string, secretFile string) *http.Client {
	return oauth2.NewClient(ctx, newTokenSource(ctx, conf, secret, secretFile))
}

func newTokenSource(ctx context.Context, conf *clientcredentials.Config, secret string, secretFile string) oauth2.TokenSource {
	// normal static secret token
	if secretFile == "" {
		conf.ClientSecret = secret
		return conf.TokenSource(ctx)
	}
	source := &fileTokenSource{
		ctx:        ctx,
		conf:       conf,
		secretFile: secretFile,
	}
	// dynamic file token source
	return oauth2.ReuseTokenSource(nil, source)
}

type fileTokenSource struct {
	ctx        context.Context
	conf       *clientcredentials.Config
	secretFile string
}

// Token refreshes the token by using a new client credentials request.
// tokens received this way do not include a refresh token
func (c *fileTokenSource) Token() (*oauth2.Token, error) {
	v := url.Values{
		"grant_type": {"client_credentials"},
	}
	if len(c.conf.Scopes) > 0 {
		v.Set("scope", strings.Join(c.conf.Scopes, " "))
	}
	for k, p := range c.conf.EndpointParams {
		// Allow grant_type to be overridden to allow interoperability with
		// non-compliant implementations.
		if _, ok := v[k]; ok && k != "grant_type" {
			return nil, fmt.Errorf("oauth2: cannot overwrite parameter %q", k)
		}
		v[k] = p
	}

	content, err := os.ReadFile(c.secretFile)
	if err != nil {
		return nil, fmt.Errorf("oauth2: cannot read token file %q: %v", c.secretFile, err)
	}

	tk, err := retrieveToken(c.ctx, c.conf.ClientID, string(content), c.conf.TokenURL, v)
	if err != nil {
		return nil, err
	}
	return tk, nil
}

func getClient(ctx context.Context) *http.Client {
	if c, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok {
		return c
	}
	return nil
}

func retrieveToken(ctx context.Context, clientID, clientSecret, tokenURL string, v url.Values) (*oauth2.Token, error) {
	client := http.DefaultClient
	if c := getClient(ctx); c != nil {
		client = c
	}

	v.Set("client_id", clientID)
	v.Set("client_secret", clientSecret)
	encoded := v.Encode()
	tj := tokenJSON{}
	_, err := MakeRequest(
		ctx,
		HTTPRequest{
			URL:    tokenURL,
			Method: "POST",
			Body:   []byte(encoded),
			OKCode: []int{200},
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
		},
		&tj,
		client,
		Backoff{
			Duration:   100 * time.Millisecond,
			MaxRetries: 2,
		},
	)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  tj.AccessToken,
		TokenType:    tj.TokenType,
		RefreshToken: tj.RefreshToken,
		Expiry:       tj.expiry(),
	}

	if token != nil && token.RefreshToken == "" {
		token.RefreshToken = v.Get("refresh_token")
	}
	return token, err
}

type tokenJSON struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (e *tokenJSON) expiry() (t time.Time) {
	if v := e.ExpiresIn; v != 0 {
		return time.Now().Add(time.Duration(v) * time.Second)
	}
	return
}

// BasicAuth returns a base64 encoded string of the user and password
func BasicAuth(user, password string) string {
	auth := fmt.Sprintf("%s:%s", user, password)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
