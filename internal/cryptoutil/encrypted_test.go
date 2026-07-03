package cryptoutil_test

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/hacerx/sf-auth-decrypt-go/internal/cryptoutil"
)

var testKey = []byte("0123456789abcdef0123456789abcdef")

func TestDecryptString(t *testing.T) {
	iv := []byte("abcdefghijkl")

	tests := []struct {
		name      string
		value     string
		key       []byte
		want      string
		wantError error
	}{
		{
			name:  "valid encrypted value",
			value: mustEncrypt(t, testKey, iv, "plain test value"),
			key:   testKey,
			want:  "plain test value",
		},
		{
			name:      "invalid format",
			value:     "not-encrypted",
			key:       testKey,
			wantError: cryptoutil.ErrInvalidFormat,
		},
		{
			name:      "wrong key",
			value:     mustEncrypt(t, testKey, iv, "plain test value"),
			key:       []byte("abcdef0123456789abcdef0123456789"),
			wantError: cryptoutil.ErrDecryptFailed,
		},
		{
			name:      "invalid key length",
			value:     mustEncrypt(t, testKey, iv, "plain test value"),
			key:       []byte("short"),
			wantError: cryptoutil.ErrDecryptFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cryptoutil.DecryptString(tt.value, tt.key)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("DecryptString() error = %v, want %v", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecryptString() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("DecryptString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecryptStringUsesJSCompatibleIVStringBytes(t *testing.T) {
	const encryptedValue = "abcdefghijkl03bfe4dca4881db21775907d6fb928533d58d53fcfe1032e7b:be9eabed5202a67a6c61f1fc7a7afbb8"

	parsed, err := cryptoutil.ParseEncryptedValue(encryptedValue)
	if err != nil {
		t.Fatalf("ParseEncryptedValue() error = %v", err)
	}
	if string(parsed.IV) != "abcdefghijkl" {
		t.Fatalf("ParseEncryptedValue() IV = %x, want UTF-8 bytes for %q", parsed.IV, "abcdefghijkl")
	}

	got, err := cryptoutil.DecryptString(encryptedValue, testKey)
	if err != nil {
		t.Fatalf("DecryptString() error = %v", err)
	}
	if got != "js-compatible plain value" {
		t.Fatalf("DecryptString() = %q, want %q", got, "js-compatible plain value")
	}
}

func TestParseEncryptedValueInvalidFormats(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "missing separator", value: "abcdefghijkl"},
		{name: "short iv", value: "000102:00112233445566778899aabbccddeeff"},
		{name: "odd ciphertext hex", value: "abcdefghijkla:00112233445566778899aabbccddeeff"},
		{name: "bad ciphertext hex", value: "abcdefghijklzz:00112233445566778899aabbccddeeff"},
		{name: "bad tag hex", value: "abcdefghijklaa:00112233445566778899aabbccddeegg"},
		{name: "short tag", value: "abcdefghijklaa:001122"},
		{name: "extra separator", value: "abcdefghijklaa:00112233445566778899aabbccddeeff:extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if cryptoutil.IsEncryptedValue(tt.value) {
				t.Fatalf("IsEncryptedValue(%q) = true, want false", tt.value)
			}
			_, err := cryptoutil.ParseEncryptedValue(tt.value)
			if !errors.Is(err, cryptoutil.ErrInvalidFormat) {
				t.Fatalf("ParseEncryptedValue() error = %v, want %v", err, cryptoutil.ErrInvalidFormat)
			}
		})
	}
}

func TestDecryptTopLevelFields(t *testing.T) {
	iv := []byte("abcdefghijkl")
	encryptedValue := mustEncrypt(t, testKey, iv, "plain test value")
	record := map[string]any{
		"encrypted": encryptedValue,
		"metadata":  "kept",
		"nested": map[string]any{
			"encrypted": encryptedValue,
		},
	}

	got, err := cryptoutil.DecryptTopLevelFields(record, testKey)
	if err != nil {
		t.Fatalf("DecryptTopLevelFields() error = %v", err)
	}
	if got["encrypted"] != "plain test value" {
		t.Fatalf("top-level encrypted value = %q, want decrypted plaintext", got["encrypted"])
	}
	if got["metadata"] != "kept" {
		t.Fatalf("metadata = %q, want preserved value", got["metadata"])
	}
	nested := got["nested"].(map[string]any)
	if nested["encrypted"] != encryptedValue {
		t.Fatalf("nested encrypted value changed; only top-level fields should be decrypted")
	}
	if record["encrypted"] != encryptedValue {
		t.Fatalf("input record was mutated")
	}
}

func TestDecryptErrorsDoNotLeakSensitiveValues(t *testing.T) {
	iv := []byte("abcdefghijkl")
	plaintext := "sensitive test plaintext"
	encryptedValue := mustEncrypt(t, testKey, iv, plaintext)
	wrongKey := []byte("abcdef0123456789abcdef0123456789")

	_, err := cryptoutil.DecryptString(encryptedValue, wrongKey)
	if !errors.Is(err, cryptoutil.ErrDecryptFailed) {
		t.Fatalf("DecryptString() error = %v, want %v", err, cryptoutil.ErrDecryptFailed)
	}
	errText := err.Error()
	for _, leaked := range []string{plaintext, encryptedValue, string(testKey), string(wrongKey)} {
		if strings.Contains(errText, leaked) {
			t.Fatalf("error leaked sensitive value %q in %q", leaked, errText)
		}
	}
}

func mustEncrypt(t *testing.T, key, iv []byte, plaintext string) string {
	t.Helper()

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		t.Fatalf("create gcm: %v", err)
	}
	sealed := gcm.Seal(nil, iv, []byte(plaintext), nil)
	ciphertext := sealed[:len(sealed)-gcm.Overhead()]
	tag := sealed[len(sealed)-gcm.Overhead():]

	return string(iv) + hex.EncodeToString(ciphertext) + ":" + hex.EncodeToString(tag)
}
