package memory

import (
	"sync"
	"time"

	"github.com/elisasre/go-common"

	"github.com/rs/zerolog/log"
)

// Memory is an implementation of Interface for memory auth.
type Memory struct {
	keys   []common.JWTKey
	keysMu sync.RWMutex
}

// NewMemory init new memory interface.
// Memory is used mainly for testing do NOT use in production.
func NewMemory() (*Memory, error) {
	m := &Memory{}
	err := m.RotateKeys()
	if err != nil {
		return nil, err
	}
	return m, nil
}

// RotateKeys rotates the jwt secrets.
func (m *Memory) RotateKeys() error {
	m.keysMu.Lock()
	defer m.keysMu.Unlock()
	start := time.Now()
	keys, err := common.GenerateNewKeys()
	if err != nil {
		return err
	}
	// private key is needed only in newest which are used to generate new tokens
	for i := range m.keys {
		m.keys[i].PrivateKey = nil
		m.keys[i].PrivateKeyAsBytes = nil
	}
	m.keys = append([]common.JWTKey{*keys}, m.keys...)

	// keep 3 latest public keys
	if len(m.keys) > 3 {
		m.keys = m.keys[0:3]
	}
	log.Info().
		Str("duration", time.Since(start).String()).
		Msg("rotate keys")
	return nil
}

// GetKeys fetch all keys from cache.
func (m *Memory) GetKeys() []common.JWTKey {
	m.keysMu.RLock()
	defer m.keysMu.RUnlock()
	data := make([]common.JWTKey, len(m.keys))
	copy(data, m.keys)
	return data
}

// GetCurrentKey fetch latest key from cache, it should have privatekey.
func (m *Memory) GetCurrentKey() common.JWTKey {
	m.keysMu.RLock()
	defer m.keysMu.RUnlock()
	return m.keys[0]
}

// RefreshKeys refresh the keys from database.
func (m *Memory) RefreshKeys(reload bool) ([]common.JWTKey, error) {
	return m.keys, nil
}
