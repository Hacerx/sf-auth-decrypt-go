# Proposal: Salesforce Auth Decryption in Go

## Intent

Create a library-first Go implementation of `sf-auth-decrypt` that reads local Salesforce auth storage, decrypts org records, and returns token-bearing records for testing, data-update, and utility workflows. The result is sensitive by default and must not be logged accidentally.

## Scope

### In Scope
- Read Salesforce local auth from both legacy `.sfdx` and newer `.sf` storage.
- Resolve aliases and org records, then decrypt encrypted top-level org fields.
- Expose a reusable Go API returning the complete decrypted org record, including tokens, with typed errors.
- Include a basic CLI as a thin adapter over the library.
- Document that decrypted records, access tokens, refresh tokens, and key material are sensitive.

### Out of Scope
- Writing, rotating, refreshing, or deleting Salesforce credentials.
- Creating a new local decryption key when the existing key is missing.
- Network calls to Salesforce APIs.
- Token logging in diagnostics, errors, examples, or tests.

## Capabilities

### New Capabilities
- `local-auth-decryption`: Read local Salesforce auth storage, resolve aliases/orgs, retrieve an existing decryption key, decrypt org records, and expose library plus CLI behavior.

### Modified Capabilities
None.

## Approach

Implement a native compatibility core with a `Client`/`Store` API, options for state directories, and a pluggable `KeyProvider`. Scan `.sfdx` and `.sf` locations, parse alias/org JSON, decrypt recognized Salesforce encrypted strings with AES-256-GCM, and preserve the complete org record for callers. Missing key, invalid format, file errors, keychain failures, and decryption failures return explicit typed errors. The CLI must be opt-in for sensitive output and must never print tokens through logs or error messages.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `openspec/specs/local-auth-decryption/spec.md` | New | Capability contract for library and CLI behavior. |
| `go.mod` | New | Initialize Go module during implementation. |
| `*.go` library package | New | Public API, storage readers, key providers, decryption, typed errors. |
| `cmd/sf-auth-decrypt-go` | New | Basic CLI wrapper over the library. |
| `README.md` | New | Usage and sensitive-data warnings. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `.sf` storage differs from `.sfdx` assumptions. | Med | Specify fixture-driven behavior and isolate storage adapters. |
| Keychain lookup differs by platform. | Med | Keep key retrieval behind `KeyProvider`; test with fakes first. |
| Tokens leak through CLI/docs/tests. | High | Require sensitive-result warnings and no token values in logs/errors/fixtures. |

## Rollback Plan

Revert the implementation PR and remove the new spec, module, library, CLI, and docs. No credential data is written, so rollback has no local auth migration step.

## Dependencies

- Existing Salesforce local auth files and existing decryption key.
- Go module initialization and test runner setup during implementation.

## Success Criteria

- [ ] Library decrypts fixture org records from `.sfdx` and `.sf` layouts.
- [ ] Missing key fails explicitly and never creates a key.
- [ ] API returns complete decrypted org records, including tokens, as sensitive caller-owned data.
- [ ] CLI exposes basic decrypt/read behavior without logging tokens.
