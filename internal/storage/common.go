package storage

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const aliasFilename = "alias.json"

var skippedJSONFiles = map[string]struct{}{
	aliasFilename: {},
	"key.json":    {},
}

type fileSystem interface {
	ReadFile(name string) ([]byte, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
}

type osFileSystem struct{}

func (osFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (osFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (osFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

type injectedFileSystem struct {
	fsys fs.FS
}

func (f injectedFileSystem) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(f.fsys, fsPath(name))
}

func (f injectedFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(f.fsys, fsPath(name))
}

func (f injectedFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(f.fsys, fsPath(root), fn)
}

func fsPath(name string) string {
	cleaned := filepath.ToSlash(filepath.Clean(name))
	if cleaned == "." {
		return cleaned
	}
	return strings.TrimPrefix(cleaned, "/")
}

func readAliases(fsys fileSystem, path string) (map[string]string, error) {
	data, err := fsys.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	aliases := make(map[string]string)
	for key, value := range raw {
		if username, ok := value.(string); ok {
			aliases[key] = username
		}
	}
	if orgs, ok := raw["orgs"].(map[string]any); ok {
		for alias, value := range orgs {
			if username, ok := value.(string); ok {
				aliases[alias] = username
			}
		}
	}

	return aliases, nil
}

func readOrgFile(fsys fileSystem, path string) (map[string]any, error) {
	data, err := fsys.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var org map[string]any
	if err := json.Unmarshal(data, &org); err != nil {
		return nil, err
	}
	return org, nil
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func isJSONOrgCandidate(entry fs.DirEntry) bool {
	if !entry.Type().IsRegular() {
		return false
	}
	name := entry.Name()
	if _, skip := skippedJSONFiles[name]; skip {
		return false
	}
	return strings.EqualFold(filepath.Ext(name), ".json")
}
