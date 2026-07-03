# Salesforce Auth Decrypt for Go

`sf-auth-decrypt-go` reads local Salesforce CLI auth files and returns decrypted org records from Go. It is library-first, read-only, and compatible with both legacy `.sfdx` state and newer `.sf` state.

Use it when you need to inspect or reuse locally authenticated Salesforce org records without shelling out to Salesforce CLI or making Salesforce network calls.

## What it does

| Capability | Behavior |
|---|---|
| Local auth discovery | Reads org records and aliases from `.sfdx` and `.sf`. |
| Alias/username resolution | Resolves a selector such as `my-alias` or `user@example.com` to one local org record. |
| Decryption | Decrypts top-level Salesforce encrypted field values using the local Salesforce CLI key. |
| CLI adapter | Prints a safe summary by default; full decrypted JSON requires `--show-secrets`. |
| Safety boundary | Does not write, refresh, rotate, delete, or create Salesforce credentials. |

## Quick path

### Use as a library

```go
package main

import (
    "context"
    "fmt"

    "github.com/hacerx/sf-auth-decrypt-go/authdecrypt"
)

func main() {
    client, err := authdecrypt.New()
    if err != nil {
        panic(err)
    }

    org, err := client.ResolveOrg(context.Background(), "my-alias")
    if err != nil {
        panic(err)
    }

    // The returned record may include decrypted token fields.
    fmt.Println(org["username"])
}
```

### Use the CLI from a checkout

```bash
go run ./cmd/sf-auth-decrypt-go my-alias
```

Default CLI output is a safe summary. It includes identifying org fields, a warning, and a list of redacted sensitive field names when present.

Print the complete decrypted org record only when you intentionally need to handle secrets:

```bash
go run ./cmd/sf-auth-decrypt-go --show-secrets my-alias
```

Install the local checkout binary if you want it on your `PATH`:

```bash
go install ./cmd/sf-auth-decrypt-go
sf-auth-decrypt-go my-alias
```

## Security first

Decrypted org records are credentials. Access tokens, refresh tokens, decrypted credential fields, and Salesforce key material can grant access to org data.

- Do not log decrypted records or token values.
- Do not paste decrypted output into issue trackers, chat, CI logs, screenshots, or public terminals.
- Use `--show-secrets` only when you intentionally need the full decrypted record.
- Treat local `.sfdx`, `.sf`, and keychain material as credentials.
- Prefer the library API when another program needs a token; avoid writing decrypted JSON to disk.

See [`SECURITY.md`](SECURITY.md) for sensitive-data handling and vulnerability reporting guidance.

## Library API

Create a client with default local Salesforce CLI locations:

```go
client, err := authdecrypt.New()
```

Resolve one org by alias or username:

```go
org, err := client.ResolveOrg(ctx, "my-alias")
```

Other public methods:

| Method | Use |
|---|---|
| `Aliases(ctx)` | Return merged alias mappings from configured storage adapters. |
| `Orgs(ctx)` | Return all discovered org records, decrypting records that contain encrypted top-level fields. |
| `OrgMap(ctx)` | Return decrypted org records keyed by username. |
| `OrgFromFile(ctx, path)` | Read and decrypt one explicit org file through configured adapters. |
| `ResolveOrg(ctx, selector)` | Resolve an alias or username and return the matching decrypted org record. |

Errors are typed and safe to print:

```go
org, err := client.ResolveOrg(ctx, selector)
if errors.Is(err, authdecrypt.ErrNotFound) {
    // No local alias or username matched selector.
}
if errors.Is(err, authdecrypt.ErrMissingKey) {
    // Local auth exists, but required decryption key material is unavailable.
}
_ = org
```

The package exposes these sentinel error kinds: `ErrNotFound`, `ErrMissingKey`, `ErrInvalidFormat`, `ErrKeychain`, `ErrDecrypt`, `ErrFile`, and `ErrConfig`.

## Configuration options

| Option | Purpose |
|---|---|
| `authdecrypt.WithHomeDir(path)` | Read `.sfdx` and `.sf` from a specific home directory. |
| `authdecrypt.WithLegacyStateDir(path)` | Read legacy `.sfdx` state from an explicit directory. |
| `authdecrypt.WithModernStateDir(path)` | Read newer `.sf` state from an explicit directory. |
| `authdecrypt.WithStorageAdapters(adapters...)` | Replace default storage adapters with caller-provided adapters. |
| `authdecrypt.WithKeyProvider(provider)` | Inject key lookup for tests or custom environments. |
| `authdecrypt.WithFileSystem(fsys)` | Read default `.sfdx` and `.sf` adapters from an injected `fs.FS`. Useful for fixtures. |

Default storage locations are based on `os.UserHomeDir()`:

```text
<home>/.sfdx
<home>/.sf
```

## CLI usage

```bash
sf-auth-decrypt-go [flags] <alias-or-username>
```

| Flag | Behavior |
|---|---|
| `--show-secrets` | Print the full decrypted org record, including sensitive values. |
| `--home <path>` | Use `<path>/.sfdx` and `<path>/.sf` as state locations. |
| `--legacy-state-dir <path>` | Use an explicit `.sfdx` directory. |
| `--modern-state-dir <path>` | Use an explicit `.sf` directory. |

Examples:

```bash
sf-auth-decrypt-go my-alias
sf-auth-decrypt-go --home /path/to/home my-alias
sf-auth-decrypt-go --legacy-state-dir /path/to/.sfdx --modern-state-dir /path/to/.sf my-alias
sf-auth-decrypt-go --show-secrets user@example.com
```

## Salesforce storage and keychain compatibility

The implementation is designed to match Salesforce CLI local auth behavior that matters for decryption:

- `.sfdx`: reads aliases and org JSON files from the legacy state directory.
- `.sf`: reads aliases and recursively discovers org JSON files from the modern state directory.
- macOS: reads the Salesforce CLI key with `/usr/bin/security`.
- Linux: reads the key with `secret-tool`, honors `SFDX_SECRET_TOOL_PATH`, retries the known transient `invalid or unencryptable secret` response, and falls back to generic `key.json` when `secret-tool` is missing.
- Windows or generic Unix mode: reads generic `.sfdx/key.json` when applicable.
- Generic Unix `key.json`: requires `0600` permissions on non-Windows systems.

More detail is in [`docs/compatibility.md`](docs/compatibility.md).

## Behavior guarantees

- No Salesforce network calls are made.
- Credential storage is read-only.
- Missing key material returns an explicit missing-key error and does not create replacement credentials.
- Safe error strings do not include token, plaintext, encrypted token, or key values.
- Top-level encrypted string fields are decrypted; nested values are preserved as-is.
- Decryption matches the source JavaScript behavior for Salesforce encrypted values: the first 12 characters are used as UTF-8 IV bytes, with ciphertext and auth tag hex-decoded.

## Testing

Run the full test suite:

```bash
go test -count=1 ./...
```

Useful local checks before sending changes:

```bash
gofmt -w authdecrypt cmd internal
go vet ./...
go test -count=1 ./...
```

The tests cover storage discovery, alias resolution, decryption behavior, no-leak error boundaries, keychain providers, read-only behavior, and CLI redaction behavior.

## Troubleshooting

| Symptom | Likely cause | What to check |
|---|---|---|
| `authdecrypt: resolve org: not_found` | No local alias or username matched the selector. | Confirm the alias or username exists in local Salesforce CLI state. |
| `authdecrypt: retrieve key: missing_key` | Org data is encrypted but key material is unavailable. | Confirm the local Salesforce CLI keychain or `.sfdx/key.json` is available for the same user profile. |
| `authdecrypt: retrieve key: keychain` | Keychain command failed or generic key file is unusable. | On Linux, verify `secret-tool` or `SFDX_SECRET_TOOL_PATH`; on Unix generic key files, verify `0600` permissions. |
| `authdecrypt: decrypt org: invalid_format` | A field looked encrypted but did not match the expected Salesforce format. | Inspect the local org JSON shape without sharing token values. |
| CLI prints only a summary | This is the safe default. | Re-run with `--show-secrets` only if you intend to handle decrypted credentials. |

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for local development and verification workflow.
