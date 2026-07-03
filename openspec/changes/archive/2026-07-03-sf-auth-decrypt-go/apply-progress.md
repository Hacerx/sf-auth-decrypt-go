# Apply Progress: sf-auth-decrypt-go

## Mode

Standard Mode. Strict TDD is disabled by OpenSpec config and cached testing capabilities because the project had no module or test runner before PR 1. PR 3 continued in Standard Mode per the orchestrator launch contract.

## Workload / PR Boundary

- Delivery strategy: chained PR slice.
- Chain strategy: stacked-to-main.
- Completed work unit: PR 1 — module, public API contracts, typed errors, encrypted-value parser/decryptor, and tests.
- Completed work unit: PR 2 — `.sfdx`/`.sf` storage adapters, key providers/fakes, selector resolution, fixture/read-only library tests, and `WithFileSystem` remediation.
- Completed Judgment Day compatibility hardening: Linux `SFDX_SECRET_TOOL_PATH` support plus missing-`secret-tool` generic fallback, Unix generic `key.json` permission validation, and committed injected `.sf` filesystem regression coverage.
- Completed Judgment Day retry fix: Linux `secret-tool` now retries the original JS transient `invalid or unencryptable secret` exit-code-1 libsecret failure up to 3 times before returning a keychain failure.
- Current completed work unit: PR 3 — CLI, README, and CLI behavior tests.
- PR 3 boundary: starts from PR 1 + PR 2; ends with all planned tasks complete and ready for final SDD verification.
- Out of scope for PR 3: unrelated packaging, release automation, credential mutation, token refresh, and network calls.

## Completed Tasks

- [x] 1.1 Create `go.mod` for `github.com/hacerx/sf-auth-decrypt-go`, unless repo metadata requires another module path.
- [x] 1.2 Create `authdecrypt/types.go`, `errors.go`, and `options.go` with `OrgRecord`, `AliasFile`, `KeyProvider`, safe typed errors, and option hooks for dirs/adapters/filesystem/key provider.
- [x] 1.3 Create `authdecrypt/client.go` and `store.go` with `New`, `Aliases`, `Orgs`, `OrgFromFile`, `OrgMap`, and `ResolveOrg` delegating through `Store`.
- [x] 2.1 Create `internal/storage/sfdx.go` to read `.sfdx/alias.json` and root org JSON while skipping `alias.json` and `key.json`.
- [x] 2.2 Create `internal/storage/sf.go` for fixture-proven `.sf` alias/org candidates, including org subdirectories.
- [x] 2.3 Create `internal/keychain/*.go` with generic `key.json` lookup, platform command providers, and fake providers; never create missing keys.
- [x] 3.1 Create `internal/cryptoutil/encrypted.go` for `iv+ciphertext:tag` detection, parsing, AES-256-GCM decrypt, and top-level-only field handling.
- [x] 3.2 Wire `authdecrypt/store.go` to resolve alias/username selectors, decrypt complete org records, preserve metadata, and keep all operations read-only.
- [x] 3.3 Create `cmd/sf-auth-decrypt-go/main.go` as a thin CLI requiring `--show-secrets` before printing decrypted records; default output must redact tokens.
- [x] 3.4 Create `README.md` with library/CLI usage and warnings that decrypted records, tokens, and key material are sensitive.
- [x] 4.1 Add table-driven tests for `internal/cryptoutil` covering valid decrypt vectors, invalid formats, decrypt failures, and non-leaking errors.
- [x] 4.2 Add `t.TempDir()` fixture tests for `.sfdx` and `.sf` discovery, alias resolution, missing selector, and unchanged storage hashes.
- [x] 4.3 Add key-provider tests proving missing-key errors and no key/credential creation.
- [x] 4.4 Add library tests for complete decrypted records, metadata preservation, token sensitivity, and typed `errors.Is`/`errors.As` behavior.
- [x] 4.5 Add CLI tests for requested-org output with `--show-secrets`, default no-token output, and failure messages without token/key values.

## Verification

- PR 1: `gofmt -w authdecrypt internal`; `go test ./...`; Judgment Day IV fix rerun: `gofmt -w internal/cryptoutil/encrypted.go internal/cryptoutil/encrypted_test.go`, `go test ./internal/cryptoutil`, `go test ./...`.
- PR 2: `gofmt -w "authdecrypt" "internal"`; `go test ./...`; `go vet ./...`.
- PR 2 warning remediation: `gofmt -w "authdecrypt/options.go" "authdecrypt/store_test.go" "internal/storage/common.go" "internal/storage/sfdx.go" "internal/storage/sf.go"`; `go test ./...`; `go vet ./...`.
- PR 3 narrow CLI test: `gofmt -w "cmd/sf-auth-decrypt-go/main.go" "cmd/sf-auth-decrypt-go/main_test.go"`; `go test -count=1 ./cmd/sf-auth-decrypt-go`.
- PR 3 full verification: `go test -count=1 ./...`; `go vet ./...`.
- Judgment Day compatibility hardening: `gofmt -w "authdecrypt/store_test.go" "internal/keychain/command.go" "internal/keychain/generic.go" "internal/keychain/keychain_test.go" "internal/keychain/generic_internal_test.go"`; `go test -count=1 ./internal/keychain ./authdecrypt`; `go test -count=1 ./...`; `go vet ./...`.
- Judgment Day retry fix: `gofmt -w "internal/keychain/command.go" "internal/keychain/keychain_test.go"`; `go test -count=1 ./internal/keychain`; `go test -count=1 ./...`; `go vet ./...`.

Result: code-level tests and vet pass after compatibility hardening and the Linux retry fix; final SDD verify report remains stale and must be rerun before archive claims.

## Notes

- PR 1 Judgment Day fix is preserved: the crypto utility treats the first 12 encrypted-value characters as UTF-8 IV/nonce bytes, matching the original JS `crypto.createDecipheriv` behavior. Ciphertext and auth tag remain hex-decoded.
- PR 2 adds default `.sfdx` and `.sf` adapters after options are applied, so `WithHomeDir`, `WithLegacyStateDir`, and `WithModernStateDir` drive default discovery while `WithStorageAdapters` remains available for custom tests/consumers.
- `.sfdx` discovery intentionally scans only root JSON files and skips `alias.json`/`key.json`, matching the source JS behavior. `.sf` discovery walks nested JSON candidates to cover org subdirectories proven by fixtures while still skipping `alias.json`/`key.json`.
- Key lookup is read-only. Generic `key.json`, command-backed Linux/macOS providers, and test fakes expose `Key` only; no provider writes or creates credentials.
- Store decryption lazily retrieves the `sfdx`/`local` key only when an org record contains decryptable top-level fields. Missing key, not found, decrypt, invalid format, file, and keychain failures remain typed and safe for `errors.Is`/`errors.As`.
- Error messages intentionally avoid embedding selector values, encrypted token values, plaintexts, or key material.
- PR 2 warning remediation wired `WithFileSystem` into the default `.sfdx` and `.sf` storage adapters. When a filesystem is injected and no custom adapters are supplied, the built-in adapters read aliases and org JSON through that filesystem; the default OS-backed behavior remains unchanged when no filesystem is injected.
- Judgment Day compatibility hardening keeps key lookup read-only while matching the original JS Linux provider selection more closely: `SFDX_SECRET_TOOL_PATH` overrides the `secret-tool` path, missing executable errors fall back to generic `key.json`, and generic Unix `key.json` files must be `0600`.
- The Linux `secret-tool` provider preserves the original JS workaround for transient libsecret failures by retrying exit code 1 with stderr containing `invalid or unencryptable secret` up to 3 times. Plain exit-code-1 missing-key responses are not retried.
- The committed injected-filesystem regression now covers both `.sfdx` and `.sf/orgs` records, replacing the previous need for a temporary `.sf` probe.
- PR 3 CLI is a thin adapter over `authdecrypt.New` and `Client.ResolveOrg`. It supports `--home`, `--legacy-state-dir`, and `--modern-state-dir`; tests inject a fake key provider through the same library option surface and never call a real keychain.
- PR 3 default CLI output prints only a safe summary with selected non-sensitive fields, a warning, and sensitive field names that were redacted. Full decrypted org JSON requires `--show-secrets`.
- CLI tests use synthetic encrypted fixture values only. They prove requested selector output with `--show-secrets`, default no-token/no-encrypted-token output, and failure messages that omit token, encrypted token, correct key, and wrong-key values.
- README documents library and CLI usage and warns that decrypted records, access tokens, refresh tokens, and key material are sensitive.

## Final Apply State

All planned PR 1, PR 2, and PR 3 tasks are complete. The change is ready for final SDD verification.
