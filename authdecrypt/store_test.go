package authdecrypt_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hacerx/sf-auth-decrypt-go/authdecrypt"
	"github.com/hacerx/sf-auth-decrypt-go/internal/keychain"
)

var testKey = []byte("0123456789abcdef0123456789abcdef")

func TestResolveOrgDecryptsCompleteRecordAndPreservesMetadata(t *testing.T) {
	home := t.TempDir()
	encryptedAccess := mustEncrypt(t, testKey, []byte("abcdefghijkl"), "decrypted access value")
	encryptedRefresh := mustEncrypt(t, testKey, []byte("mnopqrstuvwx"), "decrypted refresh value")
	writeFile(t, filepath.Join(home, ".sfdx", "alias.json"), `{"orgs":{"dev":"dev@example.com"}}`)
	writeFile(t, filepath.Join(home, ".sfdx", "dev@example.com.json"), fmt.Sprintf(`{
		"username":"dev@example.com",
		"accessToken":%q,
		"refreshToken":%q,
		"instanceUrl":"https://example.test",
		"metadata":{"kept":true}
	}`, encryptedAccess, encryptedRefresh))

	client, err := authdecrypt.New(
		authdecrypt.WithHomeDir(home),
		authdecrypt.WithKeyProvider(keychain.StaticProvider{Value: testKey}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	org, err := client.ResolveOrg(context.Background(), "dev")
	if err != nil {
		t.Fatalf("ResolveOrg() error = %v", err)
	}
	if org["username"] != "dev@example.com" {
		t.Fatalf("username = %v, want dev@example.com", org["username"])
	}
	if org["accessToken"] != "decrypted access value" {
		t.Fatalf("access token was not decrypted")
	}
	if org["refreshToken"] != "decrypted refresh value" {
		t.Fatalf("refresh token was not decrypted")
	}
	if org["instanceUrl"] != "https://example.test" {
		t.Fatalf("metadata field not preserved")
	}
	nested := org["metadata"].(map[string]any)
	if nested["kept"] != true {
		t.Fatalf("nested metadata not preserved")
	}
}

func TestResolveOrgByModernStorageUsername(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".sf", "alias.json"), `{"orgs":{"modern":"modern@example.com"}}`)
	writeFile(t, filepath.Join(home, ".sf", "orgs", "modern@example.com.json"), `{"username":"modern@example.com","orgId":"00D000000000001"}`)

	client, err := authdecrypt.New(authdecrypt.WithHomeDir(home))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	org, err := client.ResolveOrg(context.Background(), "modern@example.com")
	if err != nil {
		t.Fatalf("ResolveOrg() error = %v", err)
	}
	if org["orgId"] != "00D000000000001" {
		t.Fatalf("orgId = %v, want modern org", org["orgId"])
	}
}

func TestWithFileSystemFeedsDefaultStorageAdapters(t *testing.T) {
	fsys := fstest.MapFS{
		"home/.sfdx/alias.json":           {Data: []byte(`{"orgs":{"dev":"dev@example.com"}}`)},
		"home/.sfdx/dev@example.com.json": {Data: []byte(`{"username":"dev@example.com","instanceUrl":"https://example.test"}`)},
		"home/.sf/alias.json":             {Data: []byte(`{"orgs":{"modern":"modern@example.com"}}`)},
		"home/.sf/orgs/modern@example.com.json": {
			Data: []byte(`{"username":"modern@example.com","instanceUrl":"https://modern.example.test"}`),
		},
	}

	client, err := authdecrypt.New(
		authdecrypt.WithHomeDir("home"),
		authdecrypt.WithFileSystem(fsys),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	org, err := client.ResolveOrg(context.Background(), "dev")
	if err != nil {
		t.Fatalf("ResolveOrg() error = %v", err)
	}
	if org["username"] != "dev@example.com" {
		t.Fatalf("username = %v, want dev@example.com", org["username"])
	}

	modernOrg, err := client.ResolveOrg(context.Background(), "modern")
	if err != nil {
		t.Fatalf("ResolveOrg() modern error = %v", err)
	}
	if modernOrg["username"] != "modern@example.com" {
		t.Fatalf("modern username = %v, want modern@example.com", modernOrg["username"])
	}
}

func TestResolveOrgMissingSelectorReturnsTypedNotFound(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".sfdx", "dev@example.com.json"), `{"username":"dev@example.com"}`)
	client, err := authdecrypt.New(authdecrypt.WithHomeDir(home))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.ResolveOrg(context.Background(), "missing")
	if !errors.Is(err, authdecrypt.ErrNotFound) {
		t.Fatalf("ResolveOrg() error = %v, want ErrNotFound", err)
	}
	var authErr *authdecrypt.Error
	if !errors.As(err, &authErr) || authErr.Kind != authdecrypt.KindNotFound {
		t.Fatalf("ResolveOrg() error type = %#v, want not_found", authErr)
	}
}

func TestDecryptLeavesStorageFilesUnchanged(t *testing.T) {
	home := t.TempDir()
	encryptedAccess := mustEncrypt(t, testKey, []byte("abcdefghijkl"), "decrypted access value")
	writeFile(t, filepath.Join(home, ".sfdx", "alias.json"), `{"orgs":{"dev":"dev@example.com"}}`)
	writeFile(t, filepath.Join(home, ".sfdx", "dev@example.com.json"), fmt.Sprintf(`{"username":"dev@example.com","accessToken":%q}`, encryptedAccess))

	before := hashTree(t, filepath.Join(home, ".sfdx"))
	client, err := authdecrypt.New(
		authdecrypt.WithHomeDir(home),
		authdecrypt.WithKeyProvider(keychain.StaticProvider{Value: testKey}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := client.ResolveOrg(context.Background(), "dev"); err != nil {
		t.Fatalf("ResolveOrg() error = %v", err)
	}
	after := hashTree(t, filepath.Join(home, ".sfdx"))
	if fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("storage hashes changed after read-only decrypt")
	}
}

func TestMissingKeyReturnsTypedErrorWithoutCreatingCredentials(t *testing.T) {
	home := t.TempDir()
	encryptedAccess := mustEncrypt(t, testKey, []byte("abcdefghijkl"), "decrypted access value")
	writeFile(t, filepath.Join(home, ".sfdx", "dev@example.com.json"), fmt.Sprintf(`{"username":"dev@example.com","accessToken":%q}`, encryptedAccess))
	provider := &keychain.RecordingProvider{Err: keychain.ErrMissingKey}
	client, err := authdecrypt.New(authdecrypt.WithHomeDir(home), authdecrypt.WithKeyProvider(provider))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.ResolveOrg(context.Background(), "dev@example.com")
	if !errors.Is(err, authdecrypt.ErrMissingKey) {
		t.Fatalf("ResolveOrg() error = %v, want ErrMissingKey", err)
	}
	if provider.Calls != 1 || provider.Service != keychain.ServiceSFDX || provider.Account != keychain.AccountLocal {
		t.Fatalf("key provider calls = %d %q/%q", provider.Calls, provider.Service, provider.Account)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".sfdx", "key.json")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("missing key flow created key.json or unexpected stat error: %v", statErr)
	}
}

func TestDecryptErrorsDoNotLeakSensitiveValues(t *testing.T) {
	home := t.TempDir()
	secretPlaintext := "decrypted access value"
	encryptedAccess := mustEncrypt(t, testKey, []byte("abcdefghijkl"), secretPlaintext)
	wrongKey := []byte("abcdef0123456789abcdef0123456789")
	writeFile(t, filepath.Join(home, ".sfdx", "dev@example.com.json"), fmt.Sprintf(`{"username":"dev@example.com","accessToken":%q}`, encryptedAccess))
	client, err := authdecrypt.New(
		authdecrypt.WithHomeDir(home),
		authdecrypt.WithKeyProvider(keychain.StaticProvider{Value: wrongKey}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.ResolveOrg(context.Background(), "dev@example.com")
	if !errors.Is(err, authdecrypt.ErrDecrypt) {
		t.Fatalf("ResolveOrg() error = %v, want ErrDecrypt", err)
	}
	message := err.Error()
	for _, sensitive := range []string{secretPlaintext, encryptedAccess, string(testKey), string(wrongKey)} {
		if strings.Contains(message, sensitive) {
			t.Fatalf("error leaked sensitive value %q in %q", sensitive, message)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
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

func hashTree(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	hashes := map[string][32]byte{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		hashes[rel] = sha256.Sum256(data)
		return nil
	}); err != nil {
		t.Fatalf("hash tree: %v", err)
	}
	return hashes
}
