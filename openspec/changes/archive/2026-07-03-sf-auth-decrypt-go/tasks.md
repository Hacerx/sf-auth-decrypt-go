# Tasks: Salesforce Auth Decryption in Go

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 900-1400 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 contracts+crypto -> PR 2 storage+keys -> PR 3 CLI+docs+integration tests |
| Delivery strategy | auto-forecast |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Module, public API contracts, typed errors, encrypted-value parser/decryptor | PR 1 | Tests included; base = main. |
| 2 | `.sfdx`/`.sf` storage, key providers/fakes, selector resolution | PR 2 | Depends on PR 1; include read-only fixture tests. |
| 3 | CLI, README, scenario/integration coverage | PR 3 | Depends on PR 2; prove sensitive-output boundaries. |

## Phase 1: Module and Public Contracts

- [x] 1.1 Create `go.mod` for `github.com/hacerx/sf-auth-decrypt-go`, unless repo metadata requires another module path.
- [x] 1.2 Create `authdecrypt/types.go`, `errors.go`, and `options.go` with `OrgRecord`, `AliasFile`, `KeyProvider`, safe typed errors, and option hooks for dirs/adapters/filesystem/key provider.
- [x] 1.3 Create `authdecrypt/client.go` and `store.go` with `New`, `Aliases`, `Orgs`, `OrgFromFile`, `OrgMap`, and `ResolveOrg` delegating through `Store`.

## Phase 2: Storage and Key Boundaries

- [x] 2.1 Create `internal/storage/sfdx.go` to read `.sfdx/alias.json` and root org JSON while skipping `alias.json` and `key.json`.
- [x] 2.2 Create `internal/storage/sf.go` for fixture-proven `.sf` alias/org candidates, including org subdirectories.
- [x] 2.3 Create `internal/keychain/*.go` with generic `key.json` lookup, platform command providers, and fake providers; never create missing keys.

## Phase 3: Decryption, Resolution, and CLI

- [x] 3.1 Create `internal/cryptoutil/encrypted.go` for `iv+ciphertext:tag` detection, parsing, AES-256-GCM decrypt, and top-level-only field handling.
- [x] 3.2 Wire `authdecrypt/store.go` to resolve alias/username selectors, decrypt complete org records, preserve metadata, and keep all operations read-only.
- [x] 3.3 Create `cmd/sf-auth-decrypt-go/main.go` as a thin CLI requiring `--show-secrets` before printing decrypted records; default output must redact tokens.
- [x] 3.4 Create `README.md` with library/CLI usage and warnings that decrypted records, tokens, and key material are sensitive.

## Phase 4: Spec-Mapped Tests

- [x] 4.1 Add table-driven tests for `internal/cryptoutil` covering valid decrypt vectors, invalid formats, decrypt failures, and non-leaking errors.
- [x] 4.2 Add `t.TempDir()` fixture tests for `.sfdx` and `.sf` discovery, alias resolution, missing selector, and unchanged storage hashes.
- [x] 4.3 Add key-provider tests proving missing-key errors and no key/credential creation.
- [x] 4.4 Add library tests for complete decrypted records, metadata preservation, token sensitivity, and typed `errors.Is`/`errors.As` behavior.
- [x] 4.5 Add CLI tests for requested-org output with `--show-secrets`, default no-token output, and failure messages without token/key values.
