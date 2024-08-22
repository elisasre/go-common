package postgres

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/elisasre/go-common/v2/auth"
	"github.com/elisasre/go-common/v2/sentryutil"
	"github.com/elisasre/go-common/v2/sqlxutil"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	db     *sqlx.DB
	secret string
}

type Opt func(*DB)

var (
	ErrMissingDBConnection = fmt.Errorf("missing db connection")
	ErrMissingCryptKey     = fmt.Errorf("missing secret")
)

func New(opts ...Opt) (*DB, error) {
	d := &DB{}
	for _, opt := range opts {
		opt(d)
	}

	if d.db == nil {
		return nil, ErrMissingDBConnection
	}

	return d, nil
}

func WithSqlxDB(db *sqlx.DB) Opt {
	return func(d *DB) {
		d.db = db
	}
}

func WithSecret(secret string) Opt {
	return func(d *DB) {
		d.secret = secret
	}
}

type RawKey struct {
	sqlxutil.Model
	KID        string `db:"k_id"`
	PrivateKey []byte `db:"private_key_as_bytes"`
	PublicKey  []byte `db:"public_key_as_bytes"`
}

// ListJWTKeys lists the keys from database.
func (db *DB) ListJWTKeys(c context.Context) ([]auth.JWTKey, error) {
	span := sentryutil.MakeSpan(c, 1)
	defer span.Finish()

	const query = `
		SELECT * FROM jwt_keys
		WHERE deleted_at IS NULL
		ORDER BY id DESC`

	keys := make([]RawKey, 0)
	err := db.db.SelectContext(c, &keys, query)
	if err != nil {
		return nil, fmt.Errorf("selecting keys failed: %w", err)
	}

	response := []auth.JWTKey{}
	for _, key := range keys {
		keyAsDecrypted, err := DecryptRawKey(key, db.secret)
		if err != nil {
			return nil, err
		}
		response = append(response, keyAsDecrypted)
	}
	return response, nil
}

// RotateJWTKeys rotates the JWT keys in database.
func (db *DB) RotateJWTKeys(ctx context.Context, new auth.JWTKey) error {
	span := sentryutil.MakeSpan(ctx, 1)
	defer span.Finish()

	key, err := prepareRawKey(new, db.secret)
	if err != nil {
		return err
	}

	return sqlxutil.WithTx(ctx, db.db, func(ctx context.Context, tx *sqlx.Tx) error {
		const addQuery = `
			INSERT INTO jwt_keys (
				k_id,
				private_key_as_bytes,
				public_key_as_bytes,
				created_at,
				updated_at
			) VALUES (
				:k_id,
				:private_key_as_bytes,
				:public_key_as_bytes,
				:created_at,
				:updated_at
			)
			RETURNING id`
		if err := sqlxutil.CreateNamed(ctx, tx, key, addQuery); err != nil {
			return fmt.Errorf("adding jwt key to db failed: %w", err)
		}

		const updateQuery = `
			UPDATE jwt_keys
			SET private_key_as_bytes=NULL
			WHERE k_id != $1`
		if _, err := tx.ExecContext(ctx, updateQuery, new.KID); err != nil {
			return fmt.Errorf("resetting old jwt keys failed: %w", err)
		}

		// keep 3 latest ones
		const deleteQuery = `
			DELETE FROM jwt_keys
			WHERE id not in (
				SELECT id
				FROM jwt_keys
				ORDER BY ID DESC
				LIMIT 3
			)`
		if _, err := tx.ExecContext(ctx, deleteQuery); err != nil {
			return fmt.Errorf("deleting old jwt keys failed: %w", err)
		}
		return nil
	})
}

func DecryptRawKey(key RawKey, secret string) (auth.JWTKey, error) {
	pubBlock, _ := pem.Decode(key.PublicKey)
	pub, err := x509.ParsePKCS1PublicKey(pubBlock.Bytes)
	if err != nil {
		return auth.JWTKey{}, fmt.Errorf("unable to parse public key %w", err)
	}

	response := auth.JWTKey{
		CreatedAt: key.CreatedAt,
		KID:       key.KID,
		PublicKey: pub,
	}

	if len(key.PrivateKey) > 0 {
		privKey, err := auth.Decrypt(key.PrivateKey, keySecret(key.KID, secret))
		if err != nil {
			return auth.JWTKey{}, err
		}

		privBlock, _ := pem.Decode(privKey)
		priv, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
		if err != nil {
			return auth.JWTKey{}, fmt.Errorf("unable to parse private key %w", err)
		}
		response.PrivateKey = priv
	}
	return response, nil
}

func keySecret(kid string, secret string) string {
	return fmt.Sprintf("%s.%s", secret, kid)
}

func prepareRawKey(key auth.JWTKey, secret string) (*RawKey, error) {
	privKey, err := auth.Encrypt(auth.EncodePrivateKeyToPEM(key.PrivateKey), keySecret(key.KID, secret))
	if err != nil {
		return nil, err
	}

	return &RawKey{
		KID:        key.KID,
		PrivateKey: privKey,
		PublicKey:  auth.EncodePublicKeyToPEM(key.PublicKey),
	}, nil
}
