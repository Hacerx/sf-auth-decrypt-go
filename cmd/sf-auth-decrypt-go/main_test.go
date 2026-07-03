package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hacerx/sf-auth-decrypt-go/authdecrypt"
	"github.com/hacerx/sf-auth-decrypt-go/internal/keychain"
)

var cliTestKey = []byte("0123456789abcdef0123456789abcdef")

func TestCLIShowSecretsPrintsRequestedOrg(t *testing.T) {
	home := t.TempDir()
	accessToken := "decrypted access value"
	refreshToken := "decrypted refresh value"
	writeCLIFile(t, filepath.Join(home, ".sfdx", "alias.json"), `{"orgs":{"dev":"dev@example.com","other":"other@example.com"}}`)
	writeCLIFile(t, filepath.Join(home, ".sfdx", "dev@example.com.json"), fmt.Sprintf(`{"username":"dev@example.com","accessToken":%q,"refreshToken":%q,"instanceUrl":"https://example.test"}`, mustCLIEncrypt(t, cliTestKey, []byte("abcdefghijkl"), accessToken), mustCLIEncrypt(t, cliTestKey, []byte("mnopqrstuvwx"), refreshToken)))
	writeCLIFile(t, filepath.Join(home, ".sfdx", "other@example.com.json"), `{"username":"other@example.com","instanceUrl":"https://other.example.test"}`)

	stdout, stderr, code := runCLI(t, []string{"--home", home, "--show-secrets", "dev"}, keychain.StaticProvider{Value: cliTestKey})
	if code != 0 {
		t.Fatalf("run() exit = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"dev@example.com", accessToken, refreshToken, "https://example.test"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q in %s", want, stdout)
		}
	}
	if strings.Contains(stdout, "other@example.com") {
		t.Fatalf("stdout included non-requested org: %s", stdout)
	}
}

func TestCLIDefaultOutputDoesNotPrintTokens(t *testing.T) {
	home := t.TempDir()
	accessToken := "decrypted access value"
	refreshToken := "decrypted refresh value"
	encryptedAccess := mustCLIEncrypt(t, cliTestKey, []byte("abcdefghijkl"), accessToken)
	encryptedRefresh := mustCLIEncrypt(t, cliTestKey, []byte("mnopqrstuvwx"), refreshToken)
	writeCLIFile(t, filepath.Join(home, ".sfdx", "alias.json"), `{"orgs":{"dev":"dev@example.com"}}`)
	writeCLIFile(t, filepath.Join(home, ".sfdx", "dev@example.com.json"), fmt.Sprintf(`{"username":"dev@example.com","accessToken":%q,"refreshToken":%q,"instanceUrl":"https://example.test"}`, encryptedAccess, encryptedRefresh))

	stdout, stderr, code := runCLI(t, []string{"--home", home, "dev"}, keychain.StaticProvider{Value: cliTestKey})
	if code != 0 {
		t.Fatalf("run() exit = %d, stderr = %s", code, stderr)
	}
	for _, leaked := range []string{accessToken, refreshToken, encryptedAccess, encryptedRefresh, string(cliTestKey)} {
		if strings.Contains(stdout, leaked) {
			t.Fatalf("default stdout leaked sensitive value %q in %s", leaked, stdout)
		}
	}
	for _, want := range []string{"dev@example.com", "https://example.test", "redactedFields", "accessToken", "refreshToken"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("default stdout missing %q in %s", want, stdout)
		}
	}
}

func TestCLIFailureMessageDoesNotPrintTokenOrKeyValues(t *testing.T) {
	home := t.TempDir()
	accessToken := "decrypted access value"
	encryptedAccess := mustCLIEncrypt(t, cliTestKey, []byte("abcdefghijkl"), accessToken)
	wrongKey := []byte("abcdef0123456789abcdef0123456789")
	writeCLIFile(t, filepath.Join(home, ".sfdx", "alias.json"), `{"orgs":{"dev":"dev@example.com"}}`)
	writeCLIFile(t, filepath.Join(home, ".sfdx", "dev@example.com.json"), fmt.Sprintf(`{"username":"dev@example.com","accessToken":%q}`, encryptedAccess))

	stdout, stderr, code := runCLI(t, []string{"--home", home, "--show-secrets", "dev"}, keychain.StaticProvider{Value: wrongKey})
	if code == 0 {
		t.Fatalf("run() exit = 0, want failure")
	}
	if stdout != "" {
		t.Fatalf("failure stdout = %q, want empty", stdout)
	}
	for _, leaked := range []string{accessToken, encryptedAccess, string(cliTestKey), string(wrongKey)} {
		if strings.Contains(stderr, leaked) {
			t.Fatalf("stderr leaked sensitive value %q in %s", leaked, stderr)
		}
	}
	if !strings.Contains(stderr, "failed to resolve org") {
		t.Fatalf("stderr missing failure context: %s", stderr)
	}
}

func runCLI(t *testing.T, args []string, provider authdecrypt.KeyProvider) (string, string, int) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(args, &stdout, &stderr, runOptions{extraOptions: []authdecrypt.Option{authdecrypt.WithKeyProvider(provider)}})
	return stdout.String(), stderr.String(), code
}

func writeCLIFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func mustCLIEncrypt(t *testing.T, key, iv []byte, plaintext string) string {
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
