package mfa

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	// Overwrite time.Now() to return a fixed time.
	now = func() time.Time { return time.Date(1999, 1, 1, 1, 1, 1, 1, time.UTC) }

	const (
		secret = "QT7TBTDOLMKLRYIHV7U4JQMDSY77FYXV"
		code   = "459115" // calculated using the secret and the fixed time above
	)

	err := Validate(secret, code)
	require.NoError(t, err)
}
