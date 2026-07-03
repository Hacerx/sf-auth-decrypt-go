package authdecrypt

import "context"

const (
	LegacyStateDirName = ".sfdx"
	ModernStateDirName = ".sf"
)

type OrgRecord = map[string]any
type AliasFile = map[string]string

type KeyProvider interface {
	Key(ctx context.Context, service, account string) ([]byte, error)
}

type StorageAdapter interface {
	Aliases(ctx context.Context) (AliasFile, error)
	Orgs(ctx context.Context) ([]OrgRecord, error)
	OrgFromFile(ctx context.Context, path string) (OrgRecord, error)
}
