package auth

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/elisasre/go-common/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Token struct.
type Token struct {
	User *User
}

const (
	ScopeOpenID   = "openid"
	ScopeProfile  = "profile"
	ScopeEmail    = "email"
	ScopeGroups   = "groups"
	ScopeInternal = "internal"
)

var AllScopes = []string{ScopeOpenID, ScopeProfile, ScopeEmail, ScopeGroups, ScopeInternal}

// SignClaims contains claims that are passed to SignExpires func.
type SignClaims struct {
	Aud    string
	Exp    int64
	Iat    int64
	Issuer string
	Nonce  string
	Scopes []string
}

// SignAlgo const.
const SignAlgo = "RS256"

// NewToken constructs new token which is passed for application.
func NewToken(user *User) *Token {
	return &Token{User: user}
}

// UserJWTClaims contains struct for making and parsing jwt tokens.
type UserJWTClaims struct {
	*User
	jwt.RegisteredClaims
	Nonce string `json:"nonce,omitempty"`
}

// SignExpires makes new jwt token using expiration time and secret.
func (t *Token) SignExpires(key JWTKey, claim SignClaims) (string, error) {
	t.User.Email = common.Ptr(strings.ToLower(common.ValOrZero(t.User.Email)))
	sub := t.User.MakeSub()
	if claim.Iat == 0 {
		claim.Iat = time.Now().Unix()
	}

	if !slices.Contains(claim.Scopes, ScopeOpenID) {
		return "", fmt.Errorf("token must contain '%s' scope", ScopeOpenID)
	}

	if !slices.Contains(claim.Scopes, ScopeInternal) {
		t.User.Internal = nil
	}

	if !slices.Contains(claim.Scopes, ScopeEmail) {
		t.User.Email = nil
		t.User.EmailVerified = nil
	}

	if !slices.Contains(claim.Scopes, ScopeGroups) {
		t.User.Groups = nil
	}

	if !slices.Contains(claim.Scopes, ScopeProfile) {
		t.User.Name = nil
	}

	claims := UserJWTClaims{
		t.User,
		jwt.RegisteredClaims{
			Subject:   sub,
			Audience:  jwt.ClaimStrings{claim.Aud},
			ExpiresAt: jwt.NewNumericDate(time.Unix(claim.Exp, 0)),
			Issuer:    claim.Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Unix(claim.Iat, 0)),
		},
		claim.Nonce,
	}
	method := jwt.SigningMethodRS256
	token := jwt.Token{
		Header: map[string]interface{}{
			"typ": "JWT",
			"alg": method.Alg(),
			"kid": key.KID,
		},
		Claims: claims,
		Method: method,
	}
	if key.PrivateKey == nil {
		return "", fmt.Errorf("privatekey is nil for key %s", key.KID)
	}
	return token.SignedString(key.PrivateKey)
}

func findKidFromArray(keys []JWTKey, kid interface{}) (JWTKey, error) {
	kidAsString, ok := kid.(string)
	if !ok {
		return JWTKey{}, fmt.Errorf("not str")
	}
	for _, s := range keys {
		if s.KID == kidAsString {
			return s, nil
		}
	}
	return JWTKey{}, fmt.Errorf("could not find kid '%s'", kidAsString)
}

// ParseToken will validate jwt token and return user with jwt claims.
func ParseToken(raw string, keys []JWTKey, options ...jwt.ParserOption) (*UserJWTClaims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &UserJWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != SignAlgo {
			return nil, jwt.ErrSignatureInvalid
		}
		if val, ok := t.Header["kid"]; ok {
			key, err := findKidFromArray(keys, val)
			if err != nil {
				return nil, err
			}
			return key.PublicKey, nil
		}
		return nil, fmt.Errorf("could not find kid from headers")
	}, options...)
	if err != nil {
		return nil, err
	} else if !parsed.Valid {
		return nil, fmt.Errorf("jwt token was not valid")
	}

	claims, ok := parsed.Claims.(*UserJWTClaims)
	if !ok {
		return nil, fmt.Errorf("could not parse struct")
	}
	return claims, nil
}
