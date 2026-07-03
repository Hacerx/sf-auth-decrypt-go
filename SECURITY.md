# Security Policy

`sf-auth-decrypt-go` handles local Salesforce credential material. Treat decrypted org records, token fields, encrypted token fields, refresh tokens, and keychain material as secrets.

## Sensitive-data handling

- Do not commit real `.sfdx`, `.sf`, `key.json`, access tokens, refresh tokens, org IDs, instance URLs, or runtime output from real orgs.
- Do not paste decrypted records into issues, pull requests, chat, CI logs, screenshots, or public terminals.
- Use placeholders in examples, such as `my-alias`, `user@example.com`, and `/path/to/home`.
- Prefer injected `KeyProvider`, `StorageAdapter`, or `fs.FS` fixtures for tests.
- Keep CLI output in default summary mode unless you intentionally need `--show-secrets`.

## Reporting a vulnerability

If you find a vulnerability, please report it privately to the repository maintainer instead of opening a public issue with exploit details or secrets.

Include:

- A concise description of the issue.
- A minimal reproduction using fake tokens and fake org data.
- The affected OS and Go version when relevant.
- Whether the issue can expose, print, mutate, or create credential material.

Do not include real Salesforce tokens, org IDs, instance URLs, usernames, keychain output, or local credential files.

## Expected security boundaries

- The library and CLI are read-only against Salesforce credential storage.
- The package does not make Salesforce network calls.
- Missing key material must fail explicitly and must not create replacement credentials.
- Safe error messages must not include token, plaintext, encrypted token, or key values.
- Full decrypted CLI output requires `--show-secrets`.
