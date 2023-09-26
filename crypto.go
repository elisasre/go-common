package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" //nolint:gosec // G501: Blocklisted import crypto/md5: weak cryptographic primitive
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"time"
)

// Base64decode decodes base64 input to string.
func Base64decode(v string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	return string(data), nil
}

// Base64encode encode input to base64.
func Base64encode(v []byte) string {
	return base64.StdEncoding.EncodeToString(v)
}

func createHash(key string) string {
	hasher := md5.New() //nolint:gosec // G501: Blocklisted import crypto/md5: weak cryptographic primitive
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// Encrypt the secret input with passphrase
// source https://www.thepolyglotdeveloper.com/2018/02/encrypt-decrypt-data-golang-application-crypto-packages/
func Encrypt(data []byte, passphrase string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(createHash(passphrase)))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt the encrypted secret with passphrase.
func Decrypt(data []byte, passphrase string) ([]byte, error) {
	key := []byte(createHash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("invalid data")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open([]byte{}, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// GenerateNewKeyPair generates new private and public keys.
func GenerateNewKeyPair() (*JWTKey, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("error generating RSA private key: %w", err)
	}

	err = rsaKey.Validate()
	if err != nil {
		return nil, err
	}

	serial, err := BuildPKISerial()
	if err != nil {
		return nil, err
	}

	return &JWTKey{
		KID:        serial.String(),
		PrivateKey: rsaKey,
		PublicKey:  &rsaKey.PublicKey,
	}, nil
}

// BuildPKISerial generates random big.Int.
func BuildPKISerial() (*big.Int, error) {
	randomLimit := new(big.Int).Lsh(big.NewInt(1), 32)
	randomComponent, err := rand.Int(rand.Reader, randomLimit)
	if err != nil {
		return nil, fmt.Errorf("error generating random number: %w", err)
	}

	serial := big.NewInt(time.Now().UnixNano())
	serial.Lsh(serial, 32)
	serial.Or(serial, randomComponent)

	return serial, nil
}
