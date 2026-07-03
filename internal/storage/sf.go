package storage

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
)

type SF struct {
	dir  string
	fsys fileSystem
}

func NewSF(dir string) SF {
	return SF{dir: dir, fsys: osFileSystem{}}
}

func NewSFWithFileSystem(dir string, fsys fs.FS) SF {
	return SF{dir: dir, fsys: injectedFileSystem{fsys: fsys}}
}

func (s SF) Aliases(ctx context.Context) (map[string]string, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	return readAliases(s.fileSystem(), filepath.Join(s.dir, aliasFilename))
}

func (s SF) Orgs(ctx context.Context) ([]map[string]any, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	orgs := make([]map[string]any, 0)
	err := s.fileSystem().WalkDir(s.dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := contextErr(ctx); err != nil {
			return err
		}
		if !isJSONOrgCandidate(entry) {
			return nil
		}
		org, err := readOrgFile(s.fileSystem(), path)
		if err != nil {
			return err
		}
		orgs = append(orgs, org)
		return nil
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	return orgs, nil
}

func (s SF) OrgFromFile(ctx context.Context, path string) (map[string]any, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	return readOrgFile(s.fileSystem(), path)
}

func (s SF) fileSystem() fileSystem {
	if s.fsys == nil {
		return osFileSystem{}
	}
	return s.fsys
}
