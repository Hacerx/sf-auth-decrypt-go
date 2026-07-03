package keychain_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hacerx/sf-auth-decrypt-go/internal/keychain"
)

func TestGenericFileProviderReadsExistingKey(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "key.json"), `{"service":"sfdx","account":"local","key":"0123456789abcdef0123456789abcdef"}`)

	got, err := keychain.NewGenericFileProvider(dir).Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	if string(got) != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("Key() returned unexpected key material")
	}
}

func TestGenericFileProviderMissingKeyDoesNotCreateCredentials(t *testing.T) {
	dir := t.TempDir()
	provider := keychain.NewGenericFileProvider(dir)

	_, err := provider.Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if !errors.Is(err, keychain.ErrMissingKey) {
		t.Fatalf("Key() error = %v, want ErrMissingKey", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "key.json")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("missing key lookup created key.json or unexpected stat error: %v", statErr)
	}
}

func TestGenericFileProviderRejectsInsecureKeyPermissionsOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not enforced on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "key.json")
	writeFile(t, path, `{"service":"sfdx","account":"local","key":"0123456789abcdef0123456789abcdef"}`)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	_, err := keychain.NewGenericFileProvider(dir).Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if !errors.Is(err, keychain.ErrKeychain) {
		t.Fatalf("Key() error = %v, want ErrKeychain", err)
	}
}

func TestLinuxSecretToolProviderUsesSFDXSecretToolPath(t *testing.T) {
	secretTool := filepath.Join(t.TempDir(), "secret-tool-test")
	t.Setenv("SFDX_SECRET_TOOL_PATH", secretTool)
	var programs []string
	provider := keychain.NewLinuxSecretToolProvider(fakeRunner{
		result:   keychain.CommandResult{Stdout: "linux-key\n"},
		programs: &programs,
	})

	got, err := provider.Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	if string(got) != "linux-key" {
		t.Fatalf("Key() = %q, want linux-key", got)
	}
	if len(programs) != 1 || programs[0] != secretTool {
		t.Fatalf("program = %#v, want %q", programs, secretTool)
	}
}

func TestLinuxProviderFallsBackToGenericKeyWhenSecretToolIsMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "key.json"), `{"service":"sfdx","account":"local","key":"0123456789abcdef0123456789abcdef"}`)
	provider := keychain.NewLinuxProvider(dir, fakeRunner{err: exec.ErrNotFound})

	got, err := provider.Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	if string(got) != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("Key() returned unexpected key material")
	}
}

func TestLinuxProviderFallbackDoesNotCreateMissingGenericKey(t *testing.T) {
	dir := t.TempDir()
	provider := keychain.NewLinuxProvider(dir, fakeRunner{err: exec.ErrNotFound})

	_, err := provider.Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if !errors.Is(err, keychain.ErrMissingKey) {
		t.Fatalf("Key() error = %v, want ErrMissingKey", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "key.json")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("missing fallback lookup created key.json or unexpected stat error: %v", statErr)
	}
}

func TestLinuxSecretToolProviderRetriesTransientInvalidSecret(t *testing.T) {
	runner := &sequenceRunner{results: []keychain.CommandResult{
		{ExitCode: 1, Stderr: "invalid or unencryptable secret"},
		{ExitCode: 1, Stderr: "secret-tool: invalid or unencryptable secret"},
		{Stdout: "linux-key\n"},
	}}
	provider := keychain.NewLinuxSecretToolProvider(runner)

	got, err := provider.Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	if string(got) != "linux-key" {
		t.Fatalf("Key() = %q, want linux-key", got)
	}
	if runner.calls != 3 {
		t.Fatalf("secret-tool calls = %d, want 3", runner.calls)
	}
}

func TestLinuxSecretToolProviderStopsAfterTransientInvalidSecretRetries(t *testing.T) {
	runner := &sequenceRunner{results: []keychain.CommandResult{
		{ExitCode: 1, Stderr: "invalid or unencryptable secret"},
		{ExitCode: 1, Stderr: "invalid or unencryptable secret"},
		{ExitCode: 1, Stderr: "invalid or unencryptable secret"},
		{ExitCode: 1, Stderr: "invalid or unencryptable secret"},
	}}
	provider := keychain.NewLinuxSecretToolProvider(runner)

	_, err := provider.Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if !errors.Is(err, keychain.ErrKeychain) {
		t.Fatalf("Key() error = %v, want ErrKeychain", err)
	}
	if errors.Is(err, keychain.ErrMissingKey) {
		t.Fatalf("Key() error = %v, want non-missing-key transient failure", err)
	}
	if runner.calls != 4 {
		t.Fatalf("secret-tool calls = %d, want 4", runner.calls)
	}
}

func TestLinuxSecretToolProviderDoesNotRetryPlainMissingKey(t *testing.T) {
	runner := &sequenceRunner{results: []keychain.CommandResult{{ExitCode: 1, Stderr: "not found"}}}
	provider := keychain.NewLinuxSecretToolProvider(runner)

	_, err := provider.Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if !errors.Is(err, keychain.ErrMissingKey) {
		t.Fatalf("Key() error = %v, want ErrMissingKey", err)
	}
	if runner.calls != 1 {
		t.Fatalf("secret-tool calls = %d, want 1", runner.calls)
	}
}

func TestCommandProvidersParsePasswords(t *testing.T) {
	tests := []struct {
		name     string
		provider keychain.Provider
		want     string
	}{
		{
			name:     "linux secret-tool",
			provider: keychain.NewLinuxSecretToolProvider(fakeRunner{result: keychain.CommandResult{Stdout: "linux-key\n"}}),
			want:     "linux-key",
		},
		{
			name:     "darwin security",
			provider: keychain.NewDarwinSecurityProvider(fakeRunner{result: keychain.CommandResult{Stderr: `password: "darwin-key"`}}),
			want:     "darwin-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.provider.Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
			if err != nil {
				t.Fatalf("Key() error = %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Key() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCommandProviderMissingKey(t *testing.T) {
	provider := keychain.NewLinuxSecretToolProvider(fakeRunner{result: keychain.CommandResult{ExitCode: 1}})
	_, err := provider.Key(context.Background(), keychain.ServiceSFDX, keychain.AccountLocal)
	if !errors.Is(err, keychain.ErrMissingKey) {
		t.Fatalf("Key() error = %v, want ErrMissingKey", err)
	}
}

type fakeRunner struct {
	result   keychain.CommandResult
	err      error
	programs *[]string
}

func (r fakeRunner) Run(ctx context.Context, program string, args ...string) (keychain.CommandResult, error) {
	if r.programs != nil {
		*r.programs = append(*r.programs, program)
	}
	return r.result, r.err
}

type sequenceRunner struct {
	results []keychain.CommandResult
	calls   int
}

func (r *sequenceRunner) Run(ctx context.Context, program string, args ...string) (keychain.CommandResult, error) {
	index := r.calls
	r.calls++
	if index >= len(r.results) {
		return r.results[len(r.results)-1], nil
	}
	return r.results[index], nil
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
