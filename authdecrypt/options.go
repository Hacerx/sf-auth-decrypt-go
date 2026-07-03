package authdecrypt

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hacerx/sf-auth-decrypt-go/internal/keychain"
	"github.com/hacerx/sf-auth-decrypt-go/internal/storage"
)

type options struct {
	homeDir        string
	legacyStateDir string
	modernStateDir string
	adapters       []StorageAdapter
	fsys           fs.FS
	keyProvider    KeyProvider
}

type Option func(*options) error

func defaultOptions() (options, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return options{}, newError(KindConfig, "resolve home directory", err)
	}

	return options{homeDir: homeDir, legacyStateDir: filepath.Join(homeDir, LegacyStateDirName), modernStateDir: filepath.Join(homeDir, ModernStateDirName)}, nil
}

func (o *options) withDefaults() {
	if len(o.adapters) == 0 {
		if o.fsys == nil {
			o.adapters = []StorageAdapter{
				storage.NewSFDX(o.legacyStateDir),
				storage.NewSF(o.modernStateDir),
			}
		} else {
			o.adapters = []StorageAdapter{
				storage.NewSFDXWithFileSystem(o.legacyStateDir, o.fsys),
				storage.NewSFWithFileSystem(o.modernStateDir, o.fsys),
			}
		}
	}
	if o.keyProvider == nil {
		o.keyProvider = keychain.NewPlatformProvider(o.legacyStateDir, nil)
	}
}

func WithHomeDir(path string) Option {
	return func(o *options) error {
		if path == "" {
			return newError(KindConfig, "set home directory", nil)
		}
		o.homeDir = path
		o.legacyStateDir = filepath.Join(path, LegacyStateDirName)
		o.modernStateDir = filepath.Join(path, ModernStateDirName)
		return nil
	}
}

func WithLegacyStateDir(path string) Option {
	return func(o *options) error {
		if path == "" {
			return newError(KindConfig, "set legacy state directory", nil)
		}
		o.legacyStateDir = path
		return nil
	}
}

func WithModernStateDir(path string) Option {
	return func(o *options) error {
		if path == "" {
			return newError(KindConfig, "set modern state directory", nil)
		}
		o.modernStateDir = path
		return nil
	}
}

func WithStorageAdapters(adapters ...StorageAdapter) Option {
	return func(o *options) error {
		o.adapters = append([]StorageAdapter(nil), adapters...)
		return nil
	}
}

func WithFileSystem(fsys fs.FS) Option {
	return func(o *options) error {
		if fsys == nil {
			return newError(KindConfig, "set filesystem", nil)
		}
		o.fsys = fsys
		return nil
	}
}

func WithKeyProvider(provider KeyProvider) Option {
	return func(o *options) error {
		if provider == nil {
			return newError(KindConfig, "set key provider", nil)
		}
		o.keyProvider = provider
		return nil
	}
}
