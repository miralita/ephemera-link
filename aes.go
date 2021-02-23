package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

func Encrypt(keyStr, dataStr string) (error, []byte) {
	text := []byte(dataStr)
	key := []byte(keyStr)

	// generate a new aes cipher using our 32 byte long key
	c, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("can't create AES Cipher: %w", err), nil
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return fmt.Errorf("can't generate new GCM: %w", err), nil
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("can't populate nonce: %w", err), nil
	}

	data := gcm.Seal(nonce, nonce, text, nil)
	return nil, data
}

func Decrypt(keyStr string, data []byte) (error, string) {
	key := []byte(keyStr)

	c, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("can't create AES Cipher: %w", err), ""
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return fmt.Errorf("can't generate new GCM: %w", err), ""
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return fmt.Errorf("wrong data length"), ""
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("can't decode data: %w", err), ""
	}
	return nil, string(plaintext)
}
