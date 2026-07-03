# Contributing

This repository is a Go library plus a thin CLI for decrypting local Salesforce CLI auth records. Keep changes small, testable, and careful with secrets.

## Local setup

Requirements:

- Go 1.22 or newer.
- No Salesforce org access is required for the test suite.

Check the repository:

```bash
go test -count=1 ./...
```

## Development workflow

1. Keep generated examples and tests free of real credential data.
2. Use table-driven tests for behavior with multiple cases.
3. Use `t.TempDir()` or injected `fs.FS` fixtures for filesystem behavior.
4. Inject key providers and command runners instead of calling a real keychain in tests.
5. Run formatting and verification before review.

```bash
gofmt -w authdecrypt cmd internal
go vet ./...
go test -count=1 ./...
```

## What to test

| Change area | Expected coverage |
|---|---|
| Storage discovery | Legacy `.sfdx`, modern `.sf`, aliases, missing directories, read-only behavior. |
| Decryption | Valid Salesforce encrypted values, invalid formats, wrong keys, top-level-only behavior. |
| Key providers | Missing keys, keychain command parsing, Linux fallback/retry behavior, generic file permissions. |
| Public API | Typed errors, safe error strings, options, selector resolution. |
| CLI | Default redaction, explicit `--show-secrets`, safe failure messages. |

## Documentation rules

- Repository documentation is written in English.
- Do not claim package manager, release, or platform support that is not implemented or verified.
- Never include real tokens, org IDs, usernames, instance details, keychain output, or runtime data from a real org.
- Prefer short examples with placeholders.

## OpenSpec artifacts

OpenSpec files under `openspec/` are part of the project history and should not be excluded or deleted as cleanup.
