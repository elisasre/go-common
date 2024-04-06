package common

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/clientcredentials"
)

func ExampleBasicAuth() {
	fmt.Println(BasicAuth("username", "password"))
	// Output: dXNlcm5hbWU6cGFzc3dvcmQ=
}

func TestDecodeBasicAuth(t *testing.T) {
	out, err := Base64decode("dXNlcm5hbWU6cGFzc3dvcmQ=")
	require.NoError(t, err)
	require.Equal(t, "username:password", out)
}

func TestNewClient(t *testing.T) {
	secret := "secret"
	srv := mockSrv(secret)
	t.Cleanup(func() {
		srv.Close()
	})

	creds := &clientcredentials.Config{
		ClientID: "clientid",
		TokenURL: fmt.Sprintf("%s/oauth2/token", srv.URL),
		Scopes:   []string{"openid", "email", "groups"},
		EndpointParams: url.Values{
			"groups": []string{"test"},
		},
	}

	ctx := context.Background()
	c := NewClient(ctx, creds, secret, "")

	req, err := http.NewRequest("GET", fmt.Sprintf("%s?foo=bar", srv.URL), nil)
	require.NoError(t, err)

	resp, err := c.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewClientToken(t *testing.T) {
	secret := "tokenfile"
	srv := mockSrv(secret)
	t.Cleanup(func() {
		srv.Close()
	})

	creds := &clientcredentials.Config{
		ClientID: "clientid",
		TokenURL: fmt.Sprintf("%s/oauth2/token", srv.URL),
		Scopes:   []string{"openid", "email", "groups"},
		EndpointParams: url.Values{
			"groups": []string{"test"},
		},
	}

	ctx := context.Background()
	c := NewClient(ctx, creds, "secret", "./testdata/token")

	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s?foo=bar", srv.URL), nil)
		require.NoError(t, err)

		resp, err := c.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		time.Sleep(1 * time.Second)
	}
}

type tokenRequest struct {
	GrantType    string `form:"grant_type" json:"grant_type"`
	Scope        string `form:"scope" json:"scope"`
	ClientID     string `form:"client_id" json:"client_id"`
	ClientSecret string `form:"client_secret" json:"client_secret"`
}

func mockSrv(secret string) *httptest.Server {
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/oauth2/token", func(c *gin.Context) {
		var payload tokenRequest
		err := c.Bind(&payload)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if payload.GrantType != "client_credentials" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid grant_type"})
			return
		}

		requestedGroups := c.Request.Form["groups"]
		if len(requestedGroups) != 1 || requestedGroups[0] != "test" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid groups: %+v", requestedGroups)})
			return
		}

		if payload.Scope != "openid email groups" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid scope: %s", payload.Scope)})
			return
		}

		if payload.ClientID != "clientid" || payload.ClientSecret != secret {
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("invalid client: %+v", payload)})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token": "token",
			"token_type":   "Bearer",
			"expires_in":   1,
		})
	})
	return httptest.NewServer(r)
}
