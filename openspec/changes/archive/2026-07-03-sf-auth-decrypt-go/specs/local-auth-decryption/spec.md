# Delta for Local Auth Decryption

## ADDED Requirements

### Requirement: Local Storage Discovery

The system MUST discover local Salesforce auth data from supported `.sfdx` and `.sf` storage locations, including caller-provided state locations, without requiring Salesforce network access.

#### Scenario: Discover legacy storage

- GIVEN a readable `.sfdx` auth storage location with org records
- WHEN the caller requests available local orgs
- THEN the system returns records discovered from `.sfdx`

#### Scenario: Discover newer storage

- GIVEN a readable `.sf` auth storage location with org records
- WHEN the caller requests available local orgs
- THEN the system returns records discovered from `.sf`

### Requirement: Alias and Org Resolution

The system MUST resolve an org selector supplied as an alias or username to the matching local org record and MUST fail explicitly when no match exists.

#### Scenario: Resolve alias to org

- GIVEN an alias maps to a stored org username
- WHEN the caller requests that alias
- THEN the system returns the matching org record

#### Scenario: Missing selector

- GIVEN no alias or username matches the selector
- WHEN the caller requests that selector
- THEN the system returns an explicit not-found failure

### Requirement: Complete Decrypted Org Records

The library API MUST return the complete local org record with decryptable encrypted fields decrypted, including token fields when present, while preserving non-encrypted fields.

#### Scenario: Return complete decrypted record

- GIVEN a stored org record contains encrypted credentials and metadata
- WHEN the caller decrypts the org record with an available key
- THEN the returned record includes decrypted credentials and original metadata

### Requirement: Explicit Missing-Key Failure

The system MUST fail with an explicit missing-key failure when required key material is unavailable and MUST NOT create or store replacement key material.

#### Scenario: Missing key does not create credentials

- GIVEN encrypted org records exist but no usable decryption key exists
- WHEN the caller decrypts local org data
- THEN the system returns an explicit missing-key failure
- AND no key or credential file is created

### Requirement: Read-Only Credential Behavior

The system MUST NOT write, rotate, refresh, delete, or otherwise mutate Salesforce credential storage during discovery, resolution, or decryption.

#### Scenario: Decryption leaves storage unchanged

- GIVEN local auth storage exists before a decrypt operation
- WHEN the caller decrypts one or more org records
- THEN the local auth storage remains unchanged

### Requirement: Sensitive Output Boundaries

The system MUST treat decrypted org records, access tokens, refresh tokens, and key material as sensitive caller-owned data. Diagnostics, errors, examples, and tests MUST NOT expose token or key values.

#### Scenario: Failures do not leak secrets

- GIVEN a decrypt operation fails after reading sensitive local data
- WHEN the failure is reported
- THEN the failure message contains no token, refresh token, or key value

### Requirement: Basic CLI Behavior

The CLI MUST expose basic local read/decrypt behavior as a thin adapter over the library and MUST require explicit user intent before printing decrypted sensitive record data.

#### Scenario: CLI resolves requested org

- GIVEN local auth storage contains an org matching the requested selector
- WHEN the CLI is run with that selector and explicit sensitive-output intent
- THEN it prints the decrypted org record for that selector

#### Scenario: CLI default avoids token output

- GIVEN local auth storage contains decrypted token values
- WHEN the CLI is run without explicit sensitive-output intent
- THEN it does not print access token, refresh token, or key values
