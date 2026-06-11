package common

import (
	"testing"
)

func TestIsValidChinesePhone(t *testing.T) {
	tests := []struct {
		phone  string
		expect bool
	}{
		{"13800138000", true},
		{"15012345678", true},
		{"18612345678", true},
		{"19912345678", true},
		{"12012345678", false},
		{"11012345678", false},
		{"1380013800", false},
		{"138001380000", false},
		{"23800138000", false},
		{"", false},
		{"abc", false},
		{"1380013800a", false},
	}
	for _, tt := range tests {
		result := IsValidChinesePhone(tt.phone)
		if result != tt.expect {
			t.Errorf("IsValidChinesePhone(%q) = %v, want %v", tt.phone, result, tt.expect)
		}
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		phone  string
		expect string
	}{
		{"13800138000", "138****8000"},
		{"15012345678", "150****5678"},
		{"1234567890", "***"},
		{"123456789012", "***"},
		{"", "***"},
	}
	for _, tt := range tests {
		result := MaskPhone(tt.phone)
		if result != tt.expect {
			t.Errorf("MaskPhone(%q) = %v, want %v", tt.phone, result, tt.expect)
		}
	}
}

func TestIsPhoneLoginInput(t *testing.T) {
	if !IsPhoneLoginInput("13800138000") {
		t.Error("IsPhoneLoginInput should return true for valid phone")
	}
	if !IsPhoneLoginInput("+8613800138000") {
		t.Error("IsPhoneLoginInput should return true for +86 prefixed phone")
	}
	if !IsPhoneLoginInput("8613800138000") {
		t.Error("IsPhoneLoginInput should return true for 86 prefixed phone")
	}
	if IsPhoneLoginInput("username") {
		t.Error("IsPhoneLoginInput should return false for username")
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		phone  string
		expect string
	}{
		{"+8613800138000", "13800138000"},
		{"8613800138000", "13800138000"},
		{"13800138000", "13800138000"},
		{"+85212345678", "+85212345678"},
		{"+8612345", "+8612345"},
		{"8612345", "8612345"},
		{"", ""},
		{"+86abc", "+86abc"},
		{"8613800138000", "13800138000"},
		{"861380013800", "861380013800"},
	}
	for _, tt := range tests {
		result := NormalizePhone(tt.phone)
		if result != tt.expect {
			t.Errorf("NormalizePhone(%q) = %q, want %q", tt.phone, result, tt.expect)
		}
	}
}

func TestIsValidPhone(t *testing.T) {
	tests := []struct {
		phone  string
		expect bool
	}{
		{"+8613800138000", true},
		{"8613800138000", true},
		{"13800138000", true},
		{"+85212345678", false},
		{"12345", false},
		{"", false},
		{"+8612345", false},
	}
	for _, tt := range tests {
		result := IsValidPhone(tt.phone)
		if result != tt.expect {
			t.Errorf("IsValidPhone(%q) = %v, want %v", tt.phone, result, tt.expect)
		}
	}
}

func TestIsEmailInput(t *testing.T) {
	tests := []struct {
		input  string
		expect bool
	}{
		{"user@example.com", true},
		{"test@gmail.com", true},
		{"a.b@c.d", true},
		{"plaintext", false},
		{"13800138000", false},
		{"@missing-local.com", false},
		{"missing-domain@", false},
		{"", false},
		{"spaces in@email.com", false},
	}
	for _, tt := range tests {
		result := IsEmailInput(tt.input)
		if result != tt.expect {
			t.Errorf("IsEmailInput(%q) = %v, want %v", tt.input, result, tt.expect)
		}
	}
}

func TestIdentifyLoginInput(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"13800138000", "phone"},
		{"15012345678", "phone"},
		{"+8613800138000", "phone"},
		{"user@example.com", "email"},
		{"test@gmail.com", "email"},
		{"myusername", "username"},
		{"admin", "username"},
		{"user123", "username"},
		{"", "username"},
	}
	for _, tt := range tests {
		result := IdentifyLoginInput(tt.input)
		if result != tt.expect {
			t.Errorf("IdentifyLoginInput(%q) = %v, want %v", tt.input, result, tt.expect)
		}
	}
}

func TestEncryptDecryptPhone(t *testing.T) {
	originalCryptoSecret := CryptoSecret
	CryptoSecret = "test-secret-key-for-unit-test"
	defer func() { CryptoSecret = originalCryptoSecret }()

	phones := []string{
		"13800138000",
		"15012345678",
		"19988776655",
		"",
	}

	for _, phone := range phones {
		encrypted, err := EncryptPhone(phone)
		if err != nil {
			t.Errorf("EncryptPhone(%q) error: %v", phone, err)
			continue
		}

		if phone == "" {
			if encrypted != "" {
				t.Errorf("EncryptPhone('') should return '', got %q", encrypted)
			}
			continue
		}

		decrypted, err := DecryptPhone(encrypted)
		if err != nil {
			t.Errorf("DecryptPhone error: %v", err)
			continue
		}

		if decrypted != phone {
			t.Errorf("DecryptPhone(EncryptPhone(%q)) = %q, want %q", phone, decrypted, phone)
		}
	}
}

func TestEncryptPhoneDifferentNonce(t *testing.T) {
	originalCryptoSecret := CryptoSecret
	CryptoSecret = "test-secret-key-for-unit-test"
	defer func() { CryptoSecret = originalCryptoSecret }()

	phone := "13800138000"
	enc1, _ := EncryptPhone(phone)
	enc2, _ := EncryptPhone(phone)

	if enc1 == enc2 {
		t.Error("Two encryptions of the same phone should produce different ciphertexts (different nonce)")
	}

	dec1, _ := DecryptPhone(enc1)
	dec2, _ := DecryptPhone(enc2)

	if dec1 != phone || dec2 != phone {
		t.Error("Both decryptions should return the original phone number")
	}
}

func TestDecryptPhoneInvalidInput(t *testing.T) {
	originalCryptoSecret := CryptoSecret
	CryptoSecret = "test-secret-key-for-unit-test"
	defer func() { CryptoSecret = originalCryptoSecret }()

	_, err := DecryptPhone("invalid-base64!!!")
	if err == nil {
		t.Error("DecryptPhone should return error for invalid base64")
	}

	_, err = DecryptPhone("aQ==")
	if err == nil {
		t.Error("DecryptPhone should return error for too-short ciphertext")
	}
}

func TestHashPhone(t *testing.T) {
	originalCryptoSecret := CryptoSecret
	CryptoSecret = "test-secret-key-for-unit-test"
	defer func() { CryptoSecret = originalCryptoSecret }()

	h1 := HashPhone("13800138000")
	h2 := HashPhone("13800138000")
	if h1 != h2 {
		t.Error("HashPhone should be deterministic for same input")
	}
	if h1 == "" {
		t.Error("HashPhone should not return empty for non-empty input")
	}

	h3 := HashPhone("15012345678")
	if h1 == h3 {
		t.Error("HashPhone should produce different hashes for different phones")
	}

	if HashPhone("") != "" {
		t.Error("HashPhone should return empty for empty input")
	}
}
