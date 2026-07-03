package storage

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
)

type SFDX struct {
	dir  string
	fsys fileSystem
}

func NewSFDX(dir string) SFDX {
	return SFDX{dir: dir, fsys: osFileSystem{}}
}

func NewSFDXWithFileSystem(dir string, fsys fs.FS) SFDX {
	return SFDX{dir: dir, fsys: injectedFileSystem{fsys: fsys}}
}

func (s SFDX) Aliases(ctx context.Context) (map[string]string, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	return readAliases(s.fileSystem(), filepath.Join(s.dir, aliasFilename))
}

func (s SFDX) Orgs(ctx context.Context) ([]map[string]any, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	entries, err := s.fileSystem().ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	orgs := make([]map[string]any, 0)
	for _, entry := range entries {
		if err := contextErr(ctx); err != nil {
			return nil, err
		}
		if !isJSONOrgCandidate(entry) {
			continue
		}
		org, err := readOrgFile(s.fileSystem(), filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}

	return orgs, nil
}

func (s SFDX) OrgFromFile(ctx context.Context, path string) (map[string]any, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	return readOrgFile(s.fileSystem(), path)
}

func (s SFDX) fileSystem() fileSystem {
	if s.fsys == nil {
		return osFileSystem{}
	}
	return s.fsys
}
