package auth

import (
	"fmt"
	"strings"

	"github.com/elisasre/go-common/v2"
)

// Internal contains struct for internal non standard variables.
type Internal struct {
	Cluster     *string `json:"cluster,omitempty"`
	ChangeLimit *int    `json:"limit,omitempty"`
	MFA         *bool   `json:"mfa"`
	EmployeeID  string  `json:"employeeid,omitempty"`
}

// User contains struct for single user.
type User struct {
	Groups        []string  `json:"groups,omitempty"`
	Eid           string    `json:"custom:employeeid,omitempty"`
	Department    string    `json:"custom:department,omitempty"`
	JobTitle      string    `json:"custom:jobtitle,omitempty"`
	ImportGroups  []string  `json:"cognito:groups,omitempty"`
	Email         *string   `json:"email,omitempty"`
	EmailVerified *bool     `json:"email_verified,omitempty"`
	Name          *string   `json:"name,omitempty"`
	Internal      *Internal `json:"internal,omitempty"`
}

// MakeSub returns sub value for user.
func (u *User) MakeSub() string {
	if u == nil {
		return ""
	}
	prefix := "email"
	email := common.ValOrZero(u.Email)
	if u.IsServiceAccount() {
		prefix = "m2m"
		email = strings.ReplaceAll(email, ServiceAccountPrefix, "")
	}
	sub := fmt.Sprintf("%s:%s", prefix, email)
	if u.Internal != nil && u.Internal.EmployeeID != "" {
		sub = fmt.Sprintf("eid:%s", u.Internal.EmployeeID)
	}
	return strings.ToLower(sub)
}

// ServiceAccountPrefix email domain for service accounts.
const ServiceAccountPrefix = "@oauth2"

// IsServiceAccount returns boolean is the account service account.
func (u User) IsServiceAccount() bool {
	return strings.HasSuffix(common.ValOrZero(u.Email), ServiceAccountPrefix)
}

// TokenMFA returns state does user has MFA used in current JWT.
func (u User) TokenMFA() bool {
	if u.Internal == nil {
		return false
	}
	return common.ValOrZero(u.Internal.MFA)
}
