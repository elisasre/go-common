package memory

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRotateKeys(t *testing.T) {
	mem, err := NewMemory()
	require.NoError(t, err)
	require.Equal(t, 1, len(mem.keys))
	err = mem.RotateKeys()
	require.NoError(t, err)
	require.Equal(t, 2, len(mem.keys))
	err = mem.RotateKeys()
	require.NoError(t, err)
	require.Equal(t, 3, len(mem.keys))
	err = mem.RotateKeys()
	require.NoError(t, err)
	require.Equal(t, 3, len(mem.keys))

	ids := make([]string, 0, len(mem.keys))
	for _, key := range mem.keys {
		ids = append(ids, key.KID)
	}
	// rotate all keys
	for i := 0; i < 5; i++ {
		err = mem.RotateKeys()
		require.NoError(t, err)
	}

	// check that mem.keys does not exist in ids
	for _, key := range mem.keys {
		require.NotContains(t, ids, key.KID)
	}
	require.Equal(t, 3, len(mem.keys))
}

func TestGetAndRefresh(t *testing.T) {
	mem, err := NewMemory()
	require.NoError(t, err)
	require.Equal(t, 1, len(mem.keys))
	require.Equal(t, mem.keys[0], mem.GetCurrentKey())
	require.Equal(t, mem.keys, mem.GetKeys())
	data, err := mem.RefreshKeys(true)
	require.NoError(t, err)
	require.Equal(t, mem.keys, data)
}
