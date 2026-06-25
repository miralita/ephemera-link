package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

func Encrypt(keyStr, dataStr string) ([]byte, error) {
	text := []byte(dataStr)
	key := []byte(keyStr)

	// generate a new aes cipher using our 32 byte long key
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("can't create AES Cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, fmt.Errorf("can't generate new GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("can't populate nonce: %w", err)
	}

	data := gcm.Seal(nonce, nonce, text, nil)
	return data, nil
}

func Decrypt(keyStr string, data []byte) (string, error) {
	key := []byte(keyStr)

	c, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("can't create AES Cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", fmt.Errorf("can't generate new GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("wrong data length")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("can't decode data: %w", err)
	}
	return string(plaintext), nil
}
