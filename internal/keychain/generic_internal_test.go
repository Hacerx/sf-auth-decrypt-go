package keychain

import (
	"errors"
	"io/fs"
	"testing"
)

func TestValidateGenericFileMode(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		mode    uint32
		wantErr error
	}{
		{name: "unix accepts owner-only permissions", goos: "linux", mode: 0o600},
		{name: "unix rejects group-readable permissions", goos: "linux", mode: 0o640, wantErr: ErrKeychain},
		{name: "unix rejects world-readable permissions", goos: "darwin", mode: 0o644, wantErr: ErrKeychain},
		{name: "windows keeps permissive mode practical", goos: "windows", mode: 0o666},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGenericFileMode(fs.FileMode(tt.mode), tt.goos)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateGenericFileMode() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
