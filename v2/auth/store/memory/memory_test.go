package memory_test

import (
	"context"
	"testing"

	"github.com/elisasre/go-common/v2/auth/store/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRotateKeys(t *testing.T) {
	ctx := context.Background()
	mem, err := memory.New(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(mem.GetKeys()))
	err = mem.RotateKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, len(mem.GetKeys()))
	err = mem.RotateKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(mem.GetKeys()))
	err = mem.RotateKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(mem.GetKeys()))

	ids := make([]string, 0, len(mem.GetKeys()))
	for _, key := range mem.GetKeys() {
		ids = append(ids, key.KID)
	}
	// rotate all keys
	for i := 0; i < 5; i++ {
		err = mem.RotateKeys(ctx)
		require.NoError(t, err)
	}

	// check that old keys have been rotated
	for _, key := range mem.GetKeys() {
		assert.NotContains(t, ids, key.KID)
	}
	require.Equal(t, 3, len(mem.GetKeys()))
}

func TestGetAndRefresh(t *testing.T) {
	ctx := context.Background()
	mem, err := memory.New(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(mem.GetKeys()))
	require.Equal(t, mem.GetKeys()[0], mem.GetCurrentKey())
	require.Equal(t, mem.GetKeys(), mem.GetKeys())
	data, err := mem.RefreshKeys(ctx, true)
	require.NoError(t, err)
	require.Equal(t, mem.GetKeys(), data)
}
