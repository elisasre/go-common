package token

import (
	"fmt"
	"strings"
	"time"

	"github.com/elisasre/go-common"
	"github.com/golang-jwt/jwt"
)

// Token struct.
type Token struct {
	User *common.User
}

const (
	OpenID   = "openid"
	Profile  = "profile"
	Email    = "email"
	Groups   = "groups"
	Internal = "internal"
)

var AllScopes = []string{OpenID, Profile, Email, Groups, Internal}

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

// New constructs new token which is passed for application.
func New(user *common.User) *Token {
	return &Token{User: user}
}

// UserJWTClaims contains struct for making and parsing jwt tokens.
type UserJWTClaims struct {
	*common.User
	jwt.StandardClaims
	Nonce string `json:"nonce,omitempty"`
}

// SignExpires makes new jwt token using expiration time and secret.
func (t *Token) SignExpires(key common.JWTKey, claim SignClaims) (string, error) {
	t.User.Email = common.String(strings.ToLower(common.StringValue(t.User.Email)))
	sub := t.User.MakeSub()
	if claim.Iat == 0 {
		claim.Iat = time.Now().Unix()
	}

	if !common.Contains(claim.Scopes, OpenID) {
		return "", fmt.Errorf("token must contain '%s' scope", OpenID)
	}

	if !common.Contains(claim.Scopes, Internal) {
		t.User.Internal = nil
	}

	if !common.Contains(claim.Scopes, Email) {
		t.User.Email = nil
		t.User.EmailVerified = nil
	}

	if !common.Contains(claim.Scopes, Groups) {
		t.User.Groups = nil
	}

	if !common.Contains(claim.Scopes, Profile) {
		t.User.Name = nil
	}

	claims := UserJWTClaims{
		t.User,
		jwt.StandardClaims{
			Subject:   sub,
			Audience:  claim.Aud,
			ExpiresAt: claim.Exp,
			Issuer:    claim.Issuer,
			IssuedAt:  claim.Iat,
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
		return "", fmt.Errorf("privatekey is nil for key %d", key.ID)
	}
	return token.SignedString(key.PrivateKey)
}

func findKidFromArray(keys []common.JWTKey, kid interface{}) (common.JWTKey, error) {
	kidAsString, ok := kid.(string)
	if !ok {
		return common.JWTKey{}, fmt.Errorf("not str")
	}
	for _, s := range keys {
		if s.KID == kidAsString {
			return s, nil
		}
	}
	return common.JWTKey{}, fmt.Errorf("could not find kid '%s'", kidAsString)
}

// Parse will validate jwt token and return token.
func Parse(raw string, keys []common.JWTKey) (*UserJWTClaims, error) {
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
	})
	if err != nil {
		return nil, err
	} else if !parsed.Valid {
		return nil, jwt.ValidationError{}
	}

	claims, ok := parsed.Claims.(*UserJWTClaims)
	if !ok {
		return nil, fmt.Errorf("could not parse struct")
	}
	return claims, nil
}
