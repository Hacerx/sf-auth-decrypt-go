# Salesforce CLI Compatibility

This page records the local storage and keychain behavior that `sf-auth-decrypt-go` currently supports. It is intentionally narrow: this package decrypts existing local auth records; it does not authenticate, refresh, or call Salesforce APIs.

## Storage locations

| Salesforce CLI state | Supported behavior |
|---|---|
| Legacy `.sfdx` | Reads `alias.json` and org JSON files directly under the `.sfdx` directory. Skips non-org JSON such as `alias.json` and `key.json`. |
| Modern `.sf` | Reads `alias.json` and recursively discovers org JSON files under the `.sf` directory, including nested org storage. |
| Custom locations | `WithHomeDir`, `WithLegacyStateDir`, and `WithModernStateDir` override where state is read from. |
| Test fixtures | `WithFileSystem` feeds the default `.sfdx` and `.sf` adapters from an injected `fs.FS`. |

Missing state directories are treated as empty storage, not as fatal errors.

## Selector resolution

Selectors are matched as follows:

1. Read aliases from configured storage adapters.
2. If the selector is an alias, replace it with the mapped username.
3. Read discovered org records and key them by `username` or `userName`.
4. Return the matching org or a typed `authdecrypt.ErrNotFound` error.

## Key lookup

| Platform / mode | Supported behavior |
|---|---|
| macOS | Uses `/usr/bin/security find-generic-password -a local -s sfdx -g`. |
| Linux | Uses `secret-tool lookup user local domain sfdx`. |
| Linux override | Honors `SFDX_SECRET_TOOL_PATH` before the default `/usr/bin/secret-tool`. |
| Linux transient retry | Retries `secret-tool` exit code `1` when stderr contains `invalid or unencryptable secret`. |
| Linux missing `secret-tool` | Falls back to generic `.sfdx/key.json`. |
| Windows | Uses generic `.sfdx/key.json`. |
| Generic Unix mode | Uses generic `.sfdx/key.json` when `SF_USE_GENERIC_UNIX_KEYCHAIN=true` or `USE_GENERIC_UNIX_KEYCHAIN=true`. |

Generic Unix `key.json` files must have `0600` permissions on non-Windows systems.

## Decryption format

Encrypted Salesforce CLI values are parsed as:

```text
<12-character IV><hex ciphertext>:<32-character hex auth tag>
```

The first 12 characters are used as UTF-8 IV bytes to match the source JavaScript behavior. Ciphertext and authentication tag are hex-decoded, then decrypted with AES-256-GCM using the local Salesforce CLI key.

Only top-level string fields that look encrypted are decrypted. Nested values are preserved unchanged.

## Non-goals

- No Salesforce network calls.
- No token refresh.
- No login or OAuth flow.
- No credential creation, rotation, deletion, or migration.
- No guarantee that every historical Salesforce CLI storage variation is supported beyond the behavior covered by tests and OpenSpec verification.
