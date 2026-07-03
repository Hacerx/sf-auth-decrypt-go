## Exploration: sf-auth-decrypt-go

### Current State
The target project is an empty OpenSpec-initialized Go library project. `openspec/config.yaml` says no `go.mod`, source files, test runner, or architecture exist yet, so this change should establish the initial library shape without inventing unrelated application structure.

The inspected JavaScript source project exports a small async API:

- `getAlias(path = ~/.sfdx/alias.json)` reads and parses the alias file.
- `getOrg(path)` reads one org JSON file and decrypts encrypted top-level fields.
- `getOrgs(path = ~/.sfdx)` scans JSON files under `~/.sfdx`, skipping `alias.json` and `key.json`, then decrypts each org file.
- `getOrgsMap()` maps aliases from `alias.orgs` to decrypted org records by matching alias username to `org.username`.

Salesforce local auth behavior observed in code:

- The source project reads from `~/.sfdx` only. It defines `.sf` constants but `Global.DIR` resolves to `~/.sfdx`.
- Org files are JSON files in that directory. Non-org files are filtered only by filename and `.json` extension, not by schema.
- Alias storage is expected at `alias.json` with shape `{ "orgs": { "alias": "username" } }`.
- Encrypted values are detected heuristically as `iv+ciphertext:authTag`, where the IV is the first 12 characters used as UTF-8 bytes, the auth tag is 32 hex characters (16 bytes), ciphertext is hex, and the algorithm is AES-256-GCM.
- The keychain entry uses service `sfdx` and account `local`.
- The generic keychain fallback reads `~/.sfdx/key.json` with `service`, `account`, and `key`; Unix requires mode `0600`, Windows checks read/write access.
- Linux normally uses `/usr/bin/secret-tool lookup user local domain sfdx`, unless `SF_USE_GENERIC_UNIX_KEYCHAIN=true` is set or libsecret is unavailable.
- macOS normally uses `/usr/bin/security find-generic-password -a local -s sfdx -g`, unless `SF_USE_GENERIC_UNIX_KEYCHAIN=true` is set.
- Windows uses the generic `key.json` fallback in this source project, not Windows Credential Manager.
- If no key is found, the JS `Crypto.init()` path may create a new random key and retry. For a read-focused reusable Go library, that behavior should not be the default because silently creating a key can mask missing credentials and cannot decrypt existing org files.

Security-sensitive behavior to preserve or improve:

- Never log decrypted access tokens, refresh tokens, or key material.
- Treat decrypted org structs as sensitive values owned by the caller.
- Use typed errors rather than string-only errors so callers can distinguish missing alias file, missing org file, missing key, invalid encrypted format, keychain failure, and decryption failure.
- Keep key retrieval behind an interface so tests can avoid real keychains and consumers can inject custom providers.
- Match Node's key semantics carefully: the generated key is a 32-character hex string used as UTF-8 bytes for AES-256-GCM, not hex-decoded to 16 bytes.

### Affected Areas
- `openspec/config.yaml` — establishes the target as a new Go library with OpenSpec artifacts and no existing code/test runner.
- `README.md` in source project — documents public usage and security warning.
- `src/index.js` in source project — defines the public API shape to replicate or improve.
- `src/alias.js` in source project — defines alias file path and alias parsing behavior.
- `src/orgs.js` in source project — defines org discovery, org-file filtering, and decryption entry points.
- `src/crypto/crypto.js` in source project — defines encrypted value format, AES-256-GCM decryption, keychain service/account, key caching, and missing-key behavior.
- `src/crypto/keyChain.js` and `src/crypto/keyChainImpl.js` in source project — define platform key retrieval strategy and generic `key.json` fallback behavior.
- `src/crypto/secureBuffer.js` in source project — shows intent to reduce key exposure in memory, but Go should avoid overpromising memory secrecy.
- `src/types/alias.ts` and `src/types/orgs.ts` in source project — provide the initial data model for Go structs.

### Approaches
1. **Native compatibility library with pluggable key providers** — Build a pure Go library that reads Salesforce local auth files directly, decrypts AES-256-GCM values, and retrieves the Salesforce key through platform-specific providers behind interfaces.
   - Pros: Best fit for importable library use; exact behavior can be unit tested with `t.TempDir()` and fake key providers; no Salesforce CLI runtime dependency; public API can expose both high-level org lookup and lower-level parsing/decryption.
   - Cons: Requires careful platform work for macOS `security`, Linux `secret-tool`, generic `key.json`, and Windows behavior; must maintain compatibility with Salesforce CLI storage changes.
   - Effort: Medium

2. **Use a generic Go keyring dependency plus explicit generic-file fallback** — Use a cross-platform Go keyring package for service/account lookup and implement `~/.sfdx/key.json` fallback directly.
   - Pros: Less custom platform code; cleaner initial implementation for common keychain operations; library remains importable.
   - Cons: Risk of not matching Salesforce CLI's exact Linux/macOS secret attributes or output parsing; still needs generic file permissions, environment-variable behavior, and compatibility tests; dependency behavior may differ from Salesforce CLI expectations.
   - Effort: Medium

3. **CLI-backed provider** — Shell out to installed Salesforce tooling or source-compatible commands to obtain org auth details, then wrap results in a Go API.
   - Pros: Highest chance of matching current Salesforce CLI behavior if the CLI is installed and authenticated; less direct handling of keychain internals.
   - Cons: Poor fit for a reusable library core; external process dependency, slower calls, prompt risk, harder deterministic tests, and less control over error taxonomy/security boundaries.
   - Effort: Low initially, High operationally

### Recommendation
Use **Approach 1: native compatibility library with pluggable key providers**.

The Go project should prioritize a library-first API such as `Client` or `Store` with options for `StateDir`, `KeyProvider`, and possibly future `.sf` support. The initial public boundary should include high-level methods equivalent to the JS API (`Aliases`, `Orgs`, `Org`, `OrgMap`) plus a selector-friendly method for consumers that want an org by alias or username. Keep CLI behavior, if any, as a thin adapter outside the core package.

Recommended design direction:

- `type Client struct { ... }` with `New(options ...Option) (*Client, error)`.
- `Aliases(ctx) (AliasFile, error)`.
- `Orgs(ctx) ([]Org, error)`.
- `OrgFromFile(ctx, path string) (Org, error)`.
- `OrgMap(ctx) (map[string]Org, error)` preserving alias-to-username matching.
- `ResolveOrg(ctx, selector string) (Org, error)` where selector can be alias or username.
- `type KeyProvider interface { Key(ctx context.Context, service, account string) ([]byte, error) }`.
- `type FileSystem` or small file-reading seam for tests.
- Typed errors wrapping the underlying cause with `errors.Is`/`errors.As` support.

Testing should start once `go.mod` exists. Use table-driven tests for encrypted-format parsing, alias mapping, org-file filtering, typed error behavior, and decryption with known vectors. Use `t.TempDir()` for `~/.sfdx` fixtures and fake key providers instead of the real home directory or OS keychain. Platform keychain providers should have narrow unit tests plus optional integration tests skipped under `testing.Short()`.

### Risks
- Salesforce CLI auth storage can differ between legacy `.sfdx` and newer `.sf` layouts; the inspected JS project only uses `.sfdx` despite defining `.sf` constants.
- Keychain compatibility is sensitive: using a generic Go keyring package may not retrieve secrets stored with the same attributes as Salesforce CLI.
- The AES key must be treated as the 32-character string bytes used by Node, not hex-decoded bytes.
- The JS project decrypts only top-level JSON string fields; recursively decrypting fields in Go would be a behavior change unless explicitly specified later.
- Creating a new key on missing credentials is unsafe for a read-only library and should be avoided by default, but this intentionally diverges from the source implementation.
- Decrypted access and refresh tokens are highly sensitive; examples, errors, logs, and tests must avoid leaking real token material.

### Ready for Proposal
Yes. The proposal should define a Go library-first replication with a native decryption core, pluggable key providers, `.sfdx` compatibility as the initial supported storage layout, explicit non-goal of writing or rotating Salesforce auth credentials, and a future extension point for `.sf` storage if required.
