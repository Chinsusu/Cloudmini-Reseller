// Package crypto provides AES-256-GCM encryption for sensitive data at rest
// (e.g., proxy credentials).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// Cipher holds the AES-256-GCM cipher for encryption/decryption.
type Cipher struct {
	gcm cipher.AEAD
}

// New creates a Cipher from a 32-byte hex-encoded key
// (64 hex characters = 256 bits).
func New(hexKey string) (*Cipher, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("crypto.New: decode key: %w", err)
	}
	if len(keyBytes) != 32 {
		return nil, errors.New("crypto.New: key must be 32 bytes (64 hex chars)")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("crypto.New: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto.New: create GCM: %w", err)
	}

	return &Cipher{gcm: gcm}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext with the nonce prepended.
func (c *Cipher) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto.Encrypt: generate nonce: %w", err)
	}

	ciphertext := c.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded ciphertext produced by Encrypt.
func (c *Cipher) Decrypt(encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("crypto.Decrypt: base64 decode: %w", err)
	}

	nonceSize := c.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("crypto.Decrypt: ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto.Decrypt: decrypt: %w", err)
	}

	return plaintext, nil
}
