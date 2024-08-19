package auth_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/elisasre/go-common/v2/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleBasicAuth() {
	encoded := auth.BasicAuth("username", "password")
	out, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(out))
	// Output: username:password
}

func TestNewClient(t *testing.T) {
	secret := "secret"
	srv := mockSrv(secret)
	t.Cleanup(func() {
		srv.Close()
	})

	ctx := context.Background()
	c := auth.NewClient(ctx, &auth.ClientConfiguration{
		OAuth2: auth.OAuth2{
			ClientID: "clientid",
			TokenURL: fmt.Sprintf("%s/oauth2/token", srv.URL),
			Scopes:   []string{"openid", "email", "groups"},
			EndpointParams: url.Values{
				"groups": []string{"test"},
			},
			ClientSecret: secret,
		},
	})

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

	ctx := context.Background()
	c := auth.NewClient(ctx, &auth.ClientConfiguration{
		OAuth2: auth.OAuth2{
			ClientID: "clientid",
			TokenURL: fmt.Sprintf("%s/oauth2/token", srv.URL),
			Scopes:   []string{"openid", "email", "groups"},
			EndpointParams: url.Values{
				"groups": []string{"test"},
			},
			ClientSecretFile: "./testdata/token",
		},
	})
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

func TestBase64decode(t *testing.T) {
	encoded := "U1VDQ0VTUw=="
	decoded, err := auth.Base64decode(encoded)
	assert.Nil(t, err)
	assert.Equal(t, "SUCCESS", decoded)

	failing := "^"
	_, err = auth.Base64decode(failing)
	assert.NotNil(t, err)
}

func TestDecrypt(t *testing.T) {
	// This test makes sure that we don't brake backwards compatibility
	// by modifying both; encrypt and decrypt at the same time.
	const passwd = "mypass"
	var (
		encryptedInput = []byte{
			0xf2, 0x3, 0x92, 0x1f, 0x9b, 0xb4, 0x56, 0xc,
			0x37, 0xb4, 0x33, 0x5f, 0x1, 0xad, 0xe3, 0x66,
			0x99, 0x14, 0x3e, 0x59, 0xc9, 0x19, 0xfe, 0x3b,
			0x6f, 0x34, 0xd2, 0xd9, 0x80, 0xe7, 0x1f, 0x2f,
			0xf2, 0x15, 0xb1, 0x4, 0x3e,
		}
		expectedOutput = []byte("some data")
	)

	data, err := auth.Decrypt(encryptedInput, passwd)
	require.NoError(t, err)
	require.Equal(t, expectedOutput, data)
}

func TestEncryptDecrypt(t *testing.T) {
	const (
		input  = "supersecret"
		passwd = "testpassword"
	)

	encrypted, err := auth.Encrypt([]byte(input), passwd)
	require.NoError(t, err)
	data, err := auth.Decrypt(encrypted, passwd)
	require.NoError(t, err)
	require.Equal(t, input, string(data))
}

func FuzzEncryptDecrypt(f *testing.F) {
	f.Add([]byte("some data"), "passwd")

	f.Fuzz(func(t *testing.T, input []byte, passwd string) {
		encrypted, err := auth.Encrypt(input, passwd)
		require.NoError(t, err)
		data, err := auth.Decrypt(encrypted, passwd)
		require.NoError(t, err)
		require.Equal(t, input, data)
	})
}

func ExampleEncrypt() {
	encrypted, _ := auth.Encrypt([]byte("supersecret"), "testpassword")
	data, _ := auth.Decrypt(encrypted, "testpassword")
	fmt.Println(string(data))
	// Output: supersecret
}

func ExampleDecrypt() {
	encrypted, _ := auth.Encrypt([]byte("supersecret"), "testpassword")
	data, _ := auth.Decrypt(encrypted, "testpassword")
	fmt.Println(string(data))
	// Output: supersecret
}
