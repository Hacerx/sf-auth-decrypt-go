package keychain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

type GenericFileProvider struct {
	StateDir string
}

func NewGenericFileProvider(stateDir string) GenericFileProvider {
	return GenericFileProvider{StateDir: stateDir}
}

func (p GenericFileProvider) Key(ctx context.Context, service, account string) ([]byte, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	path := filepath.Join(p.StateDir, "key.json")
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrMissingKey
		}
		return nil, fmt.Errorf("%w", ErrKeychain)
	}
	if err := validateGenericFileMode(info.Mode().Perm(), runtime.GOOS); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrMissingKey
		}
		return nil, fmt.Errorf("%w", ErrKeychain)
	}

	var secret struct {
		Service string `json:"service"`
		Account string `json:"account"`
		Key     string `json:"key"`
	}
	if err := json.Unmarshal(data, &secret); err != nil {
		return nil, fmt.Errorf("%w", ErrKeychain)
	}
	if secret.Service != service || secret.Account != account || secret.Key == "" {
		return nil, ErrMissingKey
	}

	return []byte(secret.Key), nil
}

func validateGenericFileMode(mode fs.FileMode, goos string) error {
	if goos == "windows" {
		return nil
	}
	if mode != 0o600 {
		return fmt.Errorf("%w", ErrKeychain)
	}
	return nil
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
