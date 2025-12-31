package utiles

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// deriveKey 从密钥派生 32 字节的 AES-256 密钥
func deriveKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}

// Encrypt 使用 AES-GCM 加密明文
func Encrypt(plainText, key string) (string, error) {
	if plainText == "" {
		return "", nil
	}

	// 派生 32 字节密钥
	derivedKey := deriveKey(key)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密数据（nonce 前缀 + 密文）
	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)

	// Base64 编码返回
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt 使用 AES-GCM 解密密文
func Decrypt(cipherText, key string) (string, error) {
	if cipherText == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	// 派生 32 字节密钥
	derivedKey := deriveKey(key)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// 分离 nonce 和密文
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// 解密
	plainText, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
