package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elisasre/go-common"

	"github.com/rs/zerolog/log"
)

// Database is an implementation of Interface for database auth.
type Database struct {
	keys   []common.JWTKey
	store  common.Datastore
	keysMu sync.RWMutex
}

// NewDatabase init new database interface.
func NewDatabase(store common.Datastore) (*Database, error) {
	db := &Database{
		store: store,
	}
	keys, err := db.store.ListJWTKeys(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error ListKeys: %w", err)
	}
	if len(keys) > 0 {
		db.keysMu.Lock()
		defer db.keysMu.Unlock()
		db.keys = keys
		log.Info().Strs("keys", getKIDs(db.keys)).Msg("JWT keys loaded from database")
		return db, nil
	}
	if err := db.RotateKeys(); err != nil {
		return nil, err
	}
	return db, nil
}

func getKIDs(keys []common.JWTKey) []string {
	ids := make([]string, 0, len(keys))
	for _, k := range keys {
		ids = append(ids, k.KID)
	}
	return ids
}

// RotateKeys rotates the jwt secrets.
func (db *Database) RotateKeys() error {
	db.keysMu.Lock()
	defer db.keysMu.Unlock()
	start := time.Now()
	keys, err := common.GenerateNewKeys()
	if err != nil {
		return fmt.Errorf("error GenerateNewKeys: %w", err)
	}

	newest, err := db.store.AddJWTKey(context.Background(), *keys)
	if err != nil {
		return fmt.Errorf("error AddKeys: %w", err)
	}

	err = db.store.RotateJWTKeys(context.Background(), newest.ID)
	if err != nil {
		return err
	}

	newKeys, err := db.refreshKeys(false)
	if err != nil {
		return err
	}
	db.keys = newKeys
	log.Info().
		Strs("keys", getKIDs(db.keys)).
		Str("duration", time.Since(start).String()).
		Msg("JWT RotateKeys called")
	return nil
}

func (db *Database) refreshKeys(reload bool) ([]common.JWTKey, error) {
	keys, err := db.store.ListJWTKeys(context.Background())
	if err != nil {
		return keys, fmt.Errorf("error ListKeys: %w", err)
	}
	if reload {
		db.keys = keys
		log.Info().
			Strs("keys", getKIDs(db.keys)).
			Msg("JWT RefreshKeys called")
	}
	return keys, nil
}

// RefreshKeys refresh the keys from database.
func (db *Database) RefreshKeys(reload bool) ([]common.JWTKey, error) {
	db.keysMu.Lock()
	defer db.keysMu.Unlock()
	return db.refreshKeys(reload)
}

// GetKeys fetch all keys from cache.
func (db *Database) GetKeys() []common.JWTKey {
	db.keysMu.RLock()
	defer db.keysMu.RUnlock()
	data := make([]common.JWTKey, len(db.keys))
	copy(data, db.keys)
	return data
}

// GetCurrentKey fetch latest key from cache, it should have privatekey.
func (db *Database) GetCurrentKey() common.JWTKey {
	db.keysMu.RLock()
	defer db.keysMu.RUnlock()
	return db.keys[0]
}
