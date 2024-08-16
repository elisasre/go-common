package auth_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/elisasre/go-common/v2"
	"github.com/elisasre/go-common/v2/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestToken(t *testing.T) {
	key, err := auth.GenerateNewKeyPair()
	require.NoError(t, err)

	fullUser := auth.User{
		Name:          common.Ptr("Test User"),
		Email:         common.Ptr("email@company.com"),
		Groups:        []string{"group1", "group2"},
		EmailVerified: common.Ptr(true),
		Internal: &auth.Internal{
			EmployeeID: "123456",
			MFA:        common.Ptr(true),
		},
	}

	type testCase struct {
		name   string
		scopes []string
		err    error
		user   *auth.User
	}

	testCases := []testCase{
		{
			name:   "all scopes",
			scopes: auth.AllScopes,
			user: &auth.User{
				Name:          common.Ptr("Test User"),
				Email:         common.Ptr("email@company.com"),
				Groups:        []string{"group1", "group2"},
				EmailVerified: common.Ptr(true),
				Internal: &auth.Internal{
					EmployeeID: "123456",
					MFA:        common.Ptr(true),
				},
			},
		},
		{
			name:   "openid",
			scopes: []string{auth.ScopeOpenID},
			user:   nil,
		},
		{
			name:   "openid profile",
			scopes: []string{auth.ScopeOpenID, auth.ScopeProfile},
			user:   &auth.User{Name: common.Ptr("Test User")},
		},
		{
			name:   "openid email",
			scopes: []string{auth.ScopeOpenID, auth.ScopeEmail},
			user: &auth.User{
				Email:         common.Ptr("email@company.com"),
				EmailVerified: common.Ptr(true),
			},
		},
		{
			name:   "openid groups",
			scopes: []string{auth.ScopeOpenID, auth.ScopeGroups},
			user: &auth.User{
				Groups: []string{"group1", "group2"},
			},
		},
		{
			name:   "openid internal",
			scopes: []string{auth.ScopeOpenID, auth.ScopeInternal},
			user: &auth.User{
				Internal: &auth.Internal{
					EmployeeID: "123456",
					MFA:        common.Ptr(true),
				},
			},
		},
		{
			name:   "no openid scope",
			scopes: []string{},
			err:    fmt.Errorf("token must contain '%s' scope", auth.ScopeOpenID),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user := tc.user
			newUser := fullUser
			testUser := auth.NewToken(&newUser)
			token, err := testUser.SignExpires(key, auth.SignClaims{
				Aud:    "internal",
				Exp:    time.Now().Add(time.Hour).Unix(),
				Issuer: "http://localhost",
				Scopes: tc.scopes,
			})
			if tc.err != nil {
				require.Equal(t, tc.err, err)
			} else {
				require.NoError(t, err)
				userClaims, err := auth.ParseToken(token, []auth.JWTKey{key})
				require.NoError(t, err)
				require.Equal(t, user, userClaims.User)
			}
		})
	}
}

func TestInvalidKid(t *testing.T) {
	key, err := auth.GenerateNewKeyPair()
	require.NoError(t, err)
	key2, err := auth.GenerateNewKeyPair()
	require.NoError(t, err)

	testUser := auth.NewToken(&auth.User{})
	token, err := testUser.SignExpires(key, auth.SignClaims{
		Aud:    "internal",
		Exp:    time.Now().Add(time.Hour).Unix(),
		Issuer: "http://localhost",
		Scopes: auth.AllScopes,
	})
	require.NoError(t, err)
	_, err = auth.ParseToken(token, []auth.JWTKey{key2})
	require.Contains(t, err.Error(), fmt.Sprintf("could not find kid '%s'", key.KID))
}

func TestExpired(t *testing.T) {
	key, err := auth.GenerateNewKeyPair()
	require.NoError(t, err)

	testUser := auth.NewToken(&auth.User{})
	token, err := testUser.SignExpires(key, auth.SignClaims{
		Aud:    "internal",
		Exp:    1,
		Issuer: "http://localhost",
		Scopes: auth.AllScopes,
	})
	require.NoError(t, err)
	_, err = auth.ParseToken(token, []auth.JWTKey{key})
	require.Contains(t, err.Error(), "token is expired")
}

func TestIssuer(t *testing.T) {
	key, err := auth.GenerateNewKeyPair()
	require.NoError(t, err)

	testUser := auth.NewToken(&auth.User{})
	token, err := testUser.SignExpires(key, auth.SignClaims{
		Aud:    "internal",
		Exp:    time.Now().Add(time.Hour).Unix(),
		Issuer: "http://localhost",
		Scopes: auth.AllScopes,
	})
	require.NoError(t, err)
	_, err = auth.ParseToken(token, []auth.JWTKey{key}, jwt.WithIssuer("http://foobar"))
	require.True(t, strings.Contains(err.Error(), "token has invalid issuer"))
	_, err = auth.ParseToken(token, []auth.JWTKey{key}, jwt.WithIssuer("http://localhost"))
	require.NoError(t, err)
}
