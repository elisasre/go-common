package memory

import (
	"context"
	"time"

	"github.com/elisasre/go-common/v2/auth"
)

// Memory is im-memory storage for JWT keys which can be used as storage provider for Cache.
type Memory struct {
	keys []auth.JWTKey
}

// New creates new storage jwt key in-memory.
// Memory is meant for testing purposes, do NOT use in production.
func New() *Memory {
	return &Memory{keys: make([]auth.JWTKey, 0, 3)}
}

// GetKeys fetch all keys from cache.
func (m *Memory) ListJWTKeys(context.Context) ([]auth.JWTKey, error) {
	data := make([]auth.JWTKey, len(m.keys))
	copy(data, m.keys)
	return data, nil
}

// RotateKeys rotates the jwt secrets.
func (m *Memory) RotateJWTKeys(_ context.Context, key auth.JWTKey) error {
	key.CreatedAt = time.Now().Round(time.Millisecond).UTC()
	m.keys = append([]auth.JWTKey{key}, m.keys...)

	// private key is needed only in newest which are used to generate new tokens
	for i := range m.keys {
		if m.keys[i].KID != key.KID {
			m.keys[i].PrivateKey = nil
		}
	}

	// keep 3 latest public keys
	if len(m.keys) > 3 {
		m.keys = m.keys[0:3]
	}
	return nil
}
