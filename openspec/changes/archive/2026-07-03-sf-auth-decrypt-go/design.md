# Design: Salesforce Auth Decryption in Go

## Technical Approach

Build a library-first Go module with a thin CLI adapter. The core package reads local Salesforce auth stores, resolves aliases/usernames, decrypts Node-compatible AES-256-GCM top-level encrypted values, and returns complete org records as sensitive caller-owned data. Storage, key retrieval, and filesystem/process boundaries stay behind interfaces so `.sfdx`, `.sf`, fake fixtures, and platform keychains can evolve independently.

## Architecture Decisions

| Decision | Choice | Alternatives considered | Rationale |
|---|---|---|---|
| Public shape | `Client` over a reusable `Store` | CLI-only port; direct package functions | Matches proposal, keeps CLI thin, and gives callers testable library APIs. |
| Storage | Adapter interface with `.sfdx` and `.sf` adapters | One hard-coded scanner | The JS source only scans `.sfdx`; adapters isolate `.sf` layout differences and fixture coverage. |
| Key retrieval | `KeyProvider` interface plus platform/generic implementations | Create missing keys; hard dependency on Salesforce CLI | Read-only requirements forbid key creation, and fakes avoid real keychains in tests. |
| Decryption | Native AES-256-GCM parser matching Node behavior | Shell out to Node/Salesforce CLI; recursive decryption | Native code is importable and deterministic; source decrypts top-level strings only. |
| Errors | Typed errors with safe messages | String-only errors | Callers can branch on not found/missing key/format/keychain/decrypt failures without leaking secrets. |

## Data Flow

```text
CLI / caller -> Client -> Store -> StorageAdapter(.sfdx/.sf) -> org JSON
                      \-> KeyProvider(platform/generic/fake) -> AES-GCM decrypt
                      \-> resolver(alias|username) -> sensitive OrgRecord
```

All storage operations are read-only. No key, token, or decrypted field value is included in logs, errors, examples, or default CLI output.

## File Changes

| File | Action | Description |
|---|---|---|
| `go.mod` | Create | Initialize module, planned as `github.com/hacerx/sf-auth-decrypt-go` unless repository metadata says otherwise. |
| `authdecrypt/client.go` | Create | Public `Client`, high-level read/decrypt/resolve methods. |
| `authdecrypt/store.go` | Create | `Store`, storage adapter orchestration, selector resolution. |
| `authdecrypt/options.go` | Create | Options for home dir, explicit state dirs, adapters, key provider, filesystem, command runner. |
| `authdecrypt/types.go` | Create | `AliasFile`, `OrgRecord map[string]any`, constants for `sfdx/local`. |
| `authdecrypt/errors.go` | Create | Typed safe errors and `errors.Is`/`errors.As` support. |
| `internal/storage/sfdx.go` | Create | Reads `.sfdx/alias.json`, root org `*.json`, skips `alias.json` and `key.json`. |
| `internal/storage/sf.go` | Create | Reads newer `.sf` adapter candidates, including org subdirectories and alias files, proven by fixtures. |
| `internal/keychain/*.go` | Create | `KeyProvider`, generic `key.json`, macOS `security`, Linux `secret-tool`, Windows/generic behavior. |
| `internal/cryptoutil/encrypted.go` | Create | Encrypted-value detection, parsing, and AES-256-GCM decrypt. |
| `cmd/sf-auth-decrypt-go/main.go` | Create | Thin CLI requiring explicit `--show-secrets` before printing decrypted records. |
| `README.md` | Create | Library/CLI usage and sensitive-data warning. |

## Interfaces / Contracts

```go
type Client struct{ store *Store }
func New(opts ...Option) (*Client, error)
func (c *Client) Aliases(ctx context.Context) (AliasFile, error)
func (c *Client) Orgs(ctx context.Context) ([]OrgRecord, error)
func (c *Client) OrgFromFile(ctx context.Context, path string) (OrgRecord, error)
func (c *Client) OrgMap(ctx context.Context) (map[string]OrgRecord, error)
func (c *Client) ResolveOrg(ctx context.Context, selector string) (OrgRecord, error)

type KeyProvider interface { Key(ctx context.Context, service, account string) ([]byte, error) }
```

Encrypted strings are `iv+ciphertext:tag`: the first 12 characters are passed to AES-GCM as UTF-8 IV/nonce bytes, the remaining left-hand segment is hex ciphertext, and the tag is 32 hex characters. The key is the 32-character Salesforce key as UTF-8 bytes, not hex-decoded. Only top-level string fields are decrypted; other fields are preserved.

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | parser, decrypt vectors, typed errors, alias matching | Table-driven tests with fake key provider. |
| Filesystem | `.sfdx`/`.sf` discovery, filtering, read-only behavior | `t.TempDir()` fixtures; compare before/after file hashes. |
| Key providers | generic file permissions and command parsing | Fakes/unit tests; OS integration tests skipped in `testing.Short()`. |
| CLI | default redaction and explicit secret output | Execute CLI against fixtures; assert no token in errors/default output. |

## Migration / Rollout

No migration required. This change creates a new read-only module and must not mutate Salesforce credential storage.

## Open Questions

None.
