package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	ivLength     = 12
	tagHexLength = 32
	keyLength    = 32
)

var (
	ErrInvalidFormat = errors.New("encrypted value has invalid format")
	ErrDecryptFailed = errors.New("encrypted value decrypt failed")
)

type EncryptedValue struct {
	IV         []byte
	Ciphertext []byte
	Tag        []byte
}

func LooksEncrypted(value string) bool {
	left, right, ok := strings.Cut(value, ":")
	return ok && len(left) >= ivLength && len(right) == tagHexLength
}

func IsEncryptedValue(value string) bool {
	_, err := ParseEncryptedValue(value)
	return err == nil
}

func ParseEncryptedValue(value string) (EncryptedValue, error) {
	left, tagHex, ok := strings.Cut(value, ":")
	if !ok || strings.Contains(tagHex, ":") {
		return EncryptedValue{}, ErrInvalidFormat
	}
	if len(left) < ivLength || (len(left)-ivLength)%2 != 0 || len(tagHex) != tagHexLength {
		return EncryptedValue{}, ErrInvalidFormat
	}

	iv := []byte(left[:ivLength])
	ciphertext, err := hex.DecodeString(left[ivLength:])
	if err != nil {
		return EncryptedValue{}, ErrInvalidFormat
	}
	tag, err := hex.DecodeString(tagHex)
	if err != nil {
		return EncryptedValue{}, ErrInvalidFormat
	}

	return EncryptedValue{IV: iv, Ciphertext: ciphertext, Tag: tag}, nil
}

func DecryptString(value string, key []byte) (string, error) {
	encrypted, err := ParseEncryptedValue(value)
	if err != nil {
		return "", err
	}
	if len(key) != keyLength {
		return "", ErrDecryptFailed
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", ErrDecryptFailed
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(encrypted.IV))
	if err != nil {
		return "", ErrDecryptFailed
	}

	sealed := make([]byte, 0, len(encrypted.Ciphertext)+len(encrypted.Tag))
	sealed = append(sealed, encrypted.Ciphertext...)
	sealed = append(sealed, encrypted.Tag...)

	plaintext, err := gcm.Open(nil, encrypted.IV, sealed, nil)
	if err != nil {
		return "", ErrDecryptFailed
	}

	return string(plaintext), nil
}

func DecryptTopLevelFields(record map[string]any, key []byte) (map[string]any, error) {
	if record == nil {
		return nil, nil
	}

	decrypted := make(map[string]any, len(record))
	for field, value := range record {
		stringValue, ok := value.(string)
		if !ok || !LooksEncrypted(stringValue) {
			decrypted[field] = value
			continue
		}

		plaintext, err := DecryptString(stringValue, key)
		if err != nil {
			return nil, err
		}
		decrypted[field] = plaintext
	}

	return decrypted, nil
}
