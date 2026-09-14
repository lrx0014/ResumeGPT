package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"strings"
)

type AESGCMTokenCipher struct {
	aead cipher.AEAD
}

func NewAESGCMTokenCipher(secret string) (*AESGCMTokenCipher, error) {
	if len(strings.TrimSpace(secret)) < 32 {
		return nil, errors.New("SETTINGS_ENCRYPTION_KEY must contain at least 32 characters")
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &AESGCMTokenCipher{aead: aead}, nil
}

func (c *AESGCMTokenCipher) Encrypt(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, []byte(value), nil), nil
}

func (c *AESGCMTokenCipher) Decrypt(value []byte) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	nonceSize := c.aead.NonceSize()
	if len(value) < nonceSize {
		return "", errors.New("invalid encrypted API token")
	}
	plaintext, err := c.aead.Open(nil, value[:nonceSize], value[nonceSize:], nil)
	if err != nil {
		return "", errors.New("decrypt API token")
	}
	return string(plaintext), nil
}
