package token

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/elisasre/go-common"
	"github.com/stretchr/testify/require"
)

func TestToken(t *testing.T) {
	key, err := common.GenerateNewKeyPair()
	require.NoError(t, err)

	fullUser := common.User{
		Name:          common.String("Test User"),
		Email:         common.String("email@company.com"),
		Groups:        []string{"group1", "group2"},
		EmailVerified: common.Bool(true),
		Internal: &common.Internal{
			EmployeeID: "123456",
			MFA:        common.Bool(true),
		},
	}

	type testCase struct {
		name   string
		scopes []string
		err    error
		user   *common.User
	}

	testCases := []testCase{
		{
			name:   "all scopes",
			scopes: AllScopes,
			user: &common.User{
				Name:          common.String("Test User"),
				Email:         common.String("email@company.com"),
				Groups:        []string{"group1", "group2"},
				EmailVerified: common.Bool(true),
				Internal: &common.Internal{
					EmployeeID: "123456",
					MFA:        common.Bool(true),
				},
			},
		},
		{
			name:   "openid",
			scopes: []string{OpenID},
			user:   nil,
		},
		{
			name:   "openid profile",
			scopes: []string{OpenID, Profile},
			user:   &common.User{Name: common.String("Test User")},
		},
		{
			name:   "openid email",
			scopes: []string{OpenID, Email},
			user: &common.User{
				Email:         common.String("email@company.com"),
				EmailVerified: common.Bool(true),
			},
		},
		{
			name:   "openid groups",
			scopes: []string{OpenID, Groups},
			user: &common.User{
				Groups: []string{"group1", "group2"},
			},
		},
		{
			name:   "openid internal",
			scopes: []string{OpenID, Internal},
			user: &common.User{
				Internal: &common.Internal{
					EmployeeID: "123456",
					MFA:        common.Bool(true),
				},
			},
		},
		{
			name:   "no openid scope",
			scopes: []string{},
			err:    fmt.Errorf("token must contain '%s' scope", OpenID),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user := tc.user
			newUser := fullUser
			testUser := New(&newUser)
			token, err := testUser.SignExpires(*key, SignClaims{
				Aud:    "internal",
				Exp:    time.Now().Add(time.Hour).Unix(),
				Issuer: "http://localhost",
				Scopes: tc.scopes,
			})
			if tc.err != nil {
				require.Equal(t, tc.err, err)
			} else {
				require.NoError(t, err)
				userClaims, err := Parse(token, []common.JWTKey{*key})
				require.NoError(t, err)
				require.Equal(t, user, userClaims.User)
			}
		})
	}
}

func TestInvalidKid(t *testing.T) {
	key, err := common.GenerateNewKeyPair()
	require.NoError(t, err)
	key2, err := common.GenerateNewKeyPair()
	require.NoError(t, err)

	testUser := New(&common.User{})
	token, err := testUser.SignExpires(*key, SignClaims{
		Aud:    "internal",
		Exp:    time.Now().Add(time.Hour).Unix(),
		Issuer: "http://localhost",
		Scopes: AllScopes,
	})
	require.NoError(t, err)
	_, err = Parse(token, []common.JWTKey{*key2})
	require.Equal(t, fmt.Sprintf("could not find kid '%s'", key.KID), err.Error())
}

func TestExpired(t *testing.T) {
	key, err := common.GenerateNewKeyPair()
	require.NoError(t, err)

	testUser := New(&common.User{})
	token, err := testUser.SignExpires(*key, SignClaims{
		Aud:    "internal",
		Exp:    1,
		Issuer: "http://localhost",
		Scopes: AllScopes,
	})
	require.NoError(t, err)
	_, err = Parse(token, []common.JWTKey{*key})
	require.True(t, strings.HasPrefix(err.Error(), "token is expired by"))
}
