package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var chinesePhoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

func IsValidChinesePhone(phone string) bool {
	return chinesePhoneRegex.MatchString(phone)
}

func NormalizePhone(phone string) string {
	if strings.HasPrefix(phone, "+86") {
		rest := phone[3:]
		if IsValidChinesePhone(rest) {
			return rest
		}
		return phone
	}
	if strings.HasPrefix(phone, "86") && len(phone) == 13 {
		rest := phone[2:]
		if IsValidChinesePhone(rest) {
			return rest
		}
	}
	return phone
}

func IsValidPhone(phone string) bool {
	normalized := NormalizePhone(phone)
	return IsValidChinesePhone(normalized)
}

func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return "***"
	}
	return phone[:3] + "****" + phone[7:]
}

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func IsEmailInput(input string) bool {
	return emailRegex.MatchString(input)
}

func IsPhoneLoginInput(input string) bool {
	return IsValidPhone(input)
}

func IdentifyLoginInput(input string) string {
	if IsPhoneLoginInput(input) {
		return "phone"
	}
	if IsEmailInput(input) {
		return "email"
	}
	return "username"
}

func derivePhoneKey() []byte {
	h := hmac.New(sha256.New, []byte(CryptoSecret))
	h.Write([]byte("phone-encryption-key"))
	return h.Sum(nil)
}

func HashPhone(phone string) string {
	if phone == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(CryptoSecret))
	h.Write([]byte("phone-cache-key:" + phone))
	return hex.EncodeToString(h.Sum(nil))
}

func deriveKeyFromSecret(secret, purpose string, keyLen int) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(purpose))
	result := h.Sum(nil)
	key := make([]byte, keyLen)
	copy(key, result)
	return key
}

func EncryptPhone(phone string) (string, error) {
	if phone == "" {
		return "", nil
	}
	key := derivePhoneKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(phone), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptPhone(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	key := derivePhoneKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	return string(plaintext), nil
}
