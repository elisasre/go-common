package common

import (
	"context"
	"encoding/base64"
	"errors"
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
	ctx        context.Context //nolint:containedctx
	conf       *clientcredentials.Config
	secretFile string
	style      authStyle
}

type authStyle int

const (
	authStyleNotKnown authStyle = iota
	authStyleInHeader
	authStyleInParams
)

// Token refreshes the token by using a new client credentials request.
// tokens received this way do not include a refresh token.
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
		return nil, fmt.Errorf("oauth2: cannot read token file %q: %w", c.secretFile, err)
	}

	var tk *oauth2.Token

	switch {
	case c.style == authStyleNotKnown, c.style == authStyleInHeader:
		tk, err = retrieveToken(c.ctx, c.conf.TokenURL, c.conf.ClientID, string(content), v, authStyleInHeader)
		if err == nil {
			c.style = authStyleInHeader
			return tk, nil
		}
		if c.style == authStyleNotKnown {
			tk, err = retrieveToken(c.ctx, c.conf.TokenURL, c.conf.ClientID, string(content), v, authStyleInParams)
			if err == nil {
				c.style = authStyleInParams
				return tk, nil
			}
		}
	case c.style == authStyleInParams:
		tk, err = retrieveToken(c.ctx, c.conf.TokenURL, c.conf.ClientID, string(content), v, authStyleInParams)
		if err == nil {
			c.style = authStyleInParams
			return tk, nil
		}
	}
	return nil, err
}

func getClient(ctx context.Context) *http.Client {
	if c, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok {
		return c
	}
	return nil
}

func buildHeadersAndBody(clientID, clientSecret string, v url.Values, style authStyle) (map[string]string, url.Values) {
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	switch style {
	case authStyleInHeader, authStyleNotKnown:
		headers["Authorization"] = "Basic " + BasicAuth(url.QueryEscape(clientID), url.QueryEscape(clientSecret))
	case authStyleInParams:
		v.Set("client_id", clientID)
		v.Set("client_secret", clientSecret)
	}
	return headers, v
}

func retrieveToken(ctx context.Context, tokenURL, clientID, clientSecret string, v url.Values, style authStyle) (*oauth2.Token, error) {
	client := http.DefaultClient
	if c := getClient(ctx); c != nil {
		client = c
	}

	headers, v := buildHeadersAndBody(clientID, clientSecret, v, style)
	req := HTTPRequest{
		URL:     tokenURL,
		Method:  "POST",
		Body:    []byte(v.Encode()),
		OKCode:  []int{200},
		Headers: headers,
	}

	var tj *tokenJSON
	var err error
	tj, err = makeRequest(ctx, client, req)
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
	if token.AccessToken == "" {
		return nil, errors.New("oauth2: server response missing access_token")
	}
	return token, err
}

func makeRequest(ctx context.Context, client *http.Client, req HTTPRequest) (*tokenJSON, error) {
	// TODO: missing support for plain/form post body
	tj := &tokenJSON{}
	_, err := MakeRequest(
		ctx,
		req,
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
	return tj, nil
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

// BasicAuth returns a base64 encoded string of the user and password.
func BasicAuth(user, password string) string {
	auth := fmt.Sprintf("%s:%s", user, password)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
