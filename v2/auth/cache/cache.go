package cache

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/elisasre/go-common/v2/auth"
)

// Datastore represents required storage interface.
type Datastore interface {
	ListJWTKeys(context.Context) ([]auth.JWTKey, error)
	RotateJWTKeys(context.Context, auth.JWTKey) error
}

// Cache provides in-memory owerlay for JWT key storage with key rotation functionality.
type Cache struct {
	keys   []auth.JWTKey
	store  Datastore
	keysMu sync.RWMutex
}

// New init new database interface.
func New(ctx context.Context, store Datastore) (*Cache, error) {
	db := &Cache{
		store: store,
	}
	keys, err := db.store.ListJWTKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ListKeys: %w", err)
	}
	if len(keys) > 0 {
		db.keysMu.Lock()
		defer db.keysMu.Unlock()
		db.keys = keys
		slog.Info("JWT keys loaded from database",
			slog.Any("keys", getKIDs(db.keys)),
		)
		return db, nil
	}
	if err := db.RotateKeys(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

func getKIDs(keys []auth.JWTKey) []string {
	ids := make([]string, 0, len(keys))
	for _, k := range keys {
		ids = append(ids, k.KID)
	}
	return ids
}

// RotateKeys rotates the jwt keys.
func (db *Cache) RotateKeys(ctx context.Context) error {
	db.keysMu.Lock()
	defer db.keysMu.Unlock()
	start := time.Now()
	keys, err := auth.GenerateNewKeyPair()
	if err != nil {
		return fmt.Errorf("error GenerateNewKeyPair: %w", err)
	}

	if err := db.store.RotateJWTKeys(ctx, keys); err != nil {
		return err
	}

	if _, err := db.refreshKeys(ctx, true); err != nil {
		return err
	}

	slog.Info("JWT RotateKeys finished",
		slog.Duration("duration", time.Since(start)),
	)
	return nil
}

func (db *Cache) refreshKeys(ctx context.Context, reload bool) ([]auth.JWTKey, error) {
	keys, err := db.store.ListJWTKeys(ctx)
	if err != nil {
		return keys, fmt.Errorf("error ListKeys: %w", err)
	}
	if reload {
		db.keys = keys
		slog.Info("JWT RefreshKeys executed",
			slog.Any("keys", getKIDs(db.keys)),
		)
	}
	return keys, nil
}

// RefreshKeys refresh the keys from database.
func (db *Cache) RefreshKeys(ctx context.Context, reload bool) ([]auth.JWTKey, error) {
	db.keysMu.Lock()
	defer db.keysMu.Unlock()
	return db.refreshKeys(ctx, reload)
}

// GetKeys fetch all keys from cache.
func (db *Cache) GetKeys() []auth.JWTKey {
	db.keysMu.RLock()
	defer db.keysMu.RUnlock()
	data := make([]auth.JWTKey, len(db.keys))
	copy(data, db.keys)
	return data
}

// GetCurrentKey fetch latest key from cache, it should have privatekey.
func (db *Cache) GetCurrentKey() auth.JWTKey {
	db.keysMu.RLock()
	defer db.keysMu.RUnlock()
	return db.keys[0]
}
