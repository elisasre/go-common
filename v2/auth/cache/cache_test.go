package cache_test

import (
	"context"
	"testing"

	"github.com/elisasre/go-common/v2/auth"
	"github.com/elisasre/go-common/v2/auth/cache"
	"github.com/stretchr/testify/require"
)

type DB struct {
	keys []auth.JWTKey
}

func (store *DB) ListJWTKeys(c context.Context) ([]auth.JWTKey, error) {
	return store.keys, nil
}

func (store *DB) RotateJWTKeys(c context.Context, payload auth.JWTKey) error {
	store.keys = append(store.keys, payload)
	out := []auth.JWTKey{}
	for _, key := range store.keys {
		if key.KID != payload.KID {
			key.PrivateKey = nil
		}
		out = append(out, key)
	}
	// keep 3 latest ones
	if len(out) > 3 {
		store.keys = []auth.JWTKey{out[len(out)-3], out[len(out)-2], out[len(out)-1]}
	} else {
		store.keys = out
	}
	return nil
}

func TestRotateKeys(t *testing.T) {
	ctx := context.Background()
	store := &DB{}
	db1, err := cache.New(ctx, store)
	require.NoError(t, err)
	require.Equal(t, 1, len(db1.GetKeys()))

	db2, err := cache.New(ctx, store)
	require.NoError(t, err)
	require.Equal(t, 1, len(db2.GetKeys()))
	require.Equal(t, db1.GetKeys(), db2.GetKeys())

	err = db1.RotateKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, len(db1.GetKeys()))
	err = db1.RotateKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(db1.GetKeys()))
	err = db1.RotateKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(db1.GetKeys()))

	ids := make([]string, 0, len(db1.GetKeys()))
	for _, key := range db1.GetKeys() {
		ids = append(ids, key.KID)
	}
	// rotate all keys
	for i := 0; i < 5; i++ {
		err = db1.RotateKeys(ctx)
		require.NoError(t, err)
	}

	// check that mem.keys does not exist in ids
	for _, key := range db1.GetKeys() {
		require.NotContains(t, ids, key.KID)
	}
	require.Equal(t, 3, len(db1.GetKeys()))
}

func TestGetAndRefresh(t *testing.T) {
	ctx := context.Background()
	store := &DB{}
	db, err := cache.New(ctx, store)
	require.NoError(t, err)
	require.Equal(t, 1, len(db.GetKeys()))
	require.Equal(t, db.GetKeys()[0], db.GetCurrentKey())
	require.Equal(t, db.GetKeys(), db.GetKeys())
	data, err := db.RefreshKeys(ctx, true)
	require.NoError(t, err)
	require.Equal(t, db.GetKeys(), data)
}
