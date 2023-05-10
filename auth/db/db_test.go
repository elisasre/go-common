package database

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/elisasre/go-common"
	"github.com/stretchr/testify/require"
)

type DB struct {
	keys []common.JWTKey
}

func (store *DB) AddJWTKey(c context.Context, payload common.JWTKey) (*common.JWTKey, error) {
	id := rand.Intn(1000000)
	payload.Model.ID = uint(id)
	payload.Model.CreatedAt = time.Now()
	payload.Model.UpdatedAt = time.Now()
	store.keys = append(store.keys, payload)
	return &payload, nil
}

func (store *DB) ListJWTKeys(c context.Context) ([]common.JWTKey, error) {
	return store.keys, nil
}

func (store *DB) RotateJWTKeys(c context.Context, idx uint) error {
	out := []common.JWTKey{}
	for _, key := range store.keys {
		if key.ID != idx {
			key.PrivateKey = nil
			key.PrivateKeyAsBytes = nil
		}
		out = append(out, key)
	}
	// keep 3 latest ones
	if len(out) > 3 {
		store.keys = []common.JWTKey{out[len(out)-3], out[len(out)-2], out[len(out)-1]}
	} else {
		store.keys = out
	}
	return nil
}

func TestRotateKeys(t *testing.T) {
	store := &DB{}
	db, err := NewDatabase(store)
	require.NoError(t, err)
	require.Equal(t, 1, len(db.keys))

	db2, err := NewDatabase(store)
	require.NoError(t, err)
	require.Equal(t, 1, len(db2.keys))
	require.Equal(t, db.keys, db2.keys)

	err = db.RotateKeys()
	require.NoError(t, err)
	require.Equal(t, 2, len(db.keys))
	err = db.RotateKeys()
	require.NoError(t, err)
	require.Equal(t, 3, len(db.keys))
	err = db.RotateKeys()
	require.NoError(t, err)
	require.Equal(t, 3, len(db.keys))

	ids := make([]string, 0, len(db.keys))
	for _, key := range db.keys {
		ids = append(ids, key.KID)
	}
	// rotate all keys
	for i := 0; i < 5; i++ {
		err = db.RotateKeys()
		require.NoError(t, err)
	}

	// check that mem.keys does not exist in ids
	for _, key := range db.keys {
		require.NotContains(t, ids, key.KID)
	}
	require.Equal(t, 3, len(db.keys))
}

func TestGetAndRefresh(t *testing.T) {
	store := &DB{}
	db, err := NewDatabase(store)
	require.NoError(t, err)
	require.Equal(t, 1, len(db.keys))
	require.Equal(t, db.keys[0], db.GetCurrentKey())
	require.Equal(t, db.keys, db.GetKeys())
	data, err := db.RefreshKeys(true)
	require.NoError(t, err)
	require.Equal(t, db.keys, data)
}
