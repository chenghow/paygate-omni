// Package crypto 提供 AES-256-GCM 对称加密工具，用于敏感数据落盘加密。
// 输出格式：base64(nonce || ciphertext || gcm_tag)
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	ErrInvalidKeySize       = errors.New("crypto: key must be exactly 32 bytes for AES-256-GCM")
	ErrCiphertextTooShort   = errors.New("crypto: ciphertext is too short to be valid")
	ErrAuthenticationFailed = errors.New("crypto: GCM authentication failed, ciphertext may be tampered")
)

// Encrypt 使用 AES-256-GCM 加密 plaintext，返回 base64 编码密文。
// 每次调用生成随机 nonce，相同明文每次输出不同密文（语义安全）。
func Encrypt(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: create cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: generate nonce: %w", err)
	}
	// Seal 输出：nonce || ciphertext || auth_tag
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 解密由 Encrypt 生成的 base64 密文，返回原始明文。
// 若密文被篡改或 key 不匹配，返回 ErrAuthenticationFailed。
func Decrypt(encoded string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKeySize
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: base64 decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: create cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrCiphertextTooShort
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// 不对外暴露底层错误细节，防止侧信道信息泄露
		return "", ErrAuthenticationFailed
	}
	return string(plaintext), nil
}
