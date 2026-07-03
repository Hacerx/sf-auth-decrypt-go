package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hacerx/sf-auth-decrypt-go/internal/storage"
)

func TestSFDXDiscoveryReadsAliasesAndRootOrgJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "alias.json"), `{"orgs":{"dev":"dev@example.com"}}`)
	writeFile(t, filepath.Join(dir, "key.json"), `{"key":"not-an-org"}`)
	writeFile(t, filepath.Join(dir, "dev@example.com.json"), `{"username":"dev@example.com","instanceUrl":"https://example.test"}`)
	writeFile(t, filepath.Join(dir, "notes.txt"), `ignored`)

	adapter := storage.NewSFDX(dir)
	aliases, err := adapter.Aliases(context.Background())
	if err != nil {
		t.Fatalf("Aliases() error = %v", err)
	}
	if aliases["dev"] != "dev@example.com" {
		t.Fatalf("alias dev = %q, want dev@example.com", aliases["dev"])
	}

	orgs, err := adapter.Orgs(context.Background())
	if err != nil {
		t.Fatalf("Orgs() error = %v", err)
	}
	if len(orgs) != 1 {
		t.Fatalf("Orgs() len = %d, want 1", len(orgs))
	}
	if orgs[0]["username"] != "dev@example.com" {
		t.Fatalf("username = %v, want dev@example.com", orgs[0]["username"])
	}
}

func TestSFDiscoveryReadsRootAndNestedOrgJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "alias.json"), `{"orgs":{"root":"root@example.com","nested":"nested@example.com"}}`)
	writeFile(t, filepath.Join(dir, "root@example.com.json"), `{"username":"root@example.com"}`)
	writeFile(t, filepath.Join(dir, "orgs", "nested@example.com.json"), `{"username":"nested@example.com"}`)
	writeFile(t, filepath.Join(dir, "orgs", "key.json"), `{"username":"ignored@example.com"}`)

	adapter := storage.NewSF(dir)
	orgs, err := adapter.Orgs(context.Background())
	if err != nil {
		t.Fatalf("Orgs() error = %v", err)
	}

	usernames := map[string]bool{}
	for _, org := range orgs {
		username, _ := org["username"].(string)
		usernames[username] = true
	}
	for _, username := range []string{"root@example.com", "nested@example.com"} {
		if !usernames[username] {
			t.Fatalf("expected discovered username %q in %#v", username, usernames)
		}
	}
	if usernames["ignored@example.com"] {
		t.Fatalf("key.json should not be treated as an org record")
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
