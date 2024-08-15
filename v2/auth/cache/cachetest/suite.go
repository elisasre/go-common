package cachetest

import (
	"context"
	"testing"

	"github.com/elisasre/go-common/v2/auth/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RunSuite(t *testing.T, store cache.Datastore) {
	c, err := cache.New(context.Background(), store)
	require.NoError(t, err)

	require.Equal(t, 1, len(c.GetKeys()))
	err = c.RotateKeys(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, len(c.GetKeys()))
	err = c.RotateKeys(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, len(c.GetKeys()))
	err = c.RotateKeys(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, len(c.GetKeys()))

	ids := make([]string, 0, len(c.GetKeys()))
	for _, key := range c.GetKeys() {
		ids = append(ids, key.KID)
	}

	// rotate all keys
	for i := 0; i < 5; i++ {
		err = c.RotateKeys(context.Background())
		require.NoError(t, err)
	}

	// check that old keys have been rotated
	for _, key := range c.GetKeys() {
		assert.NotContains(t, ids, key.KID)
	}
	require.Equal(t, 3, len(c.GetKeys()))
}
