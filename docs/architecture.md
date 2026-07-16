# Architecture

## Components

```text
React/Vite web UI and packaged Android UI
             |
             | HTTPS JSON API
             v
Gin handlers -> middleware -> repository -> PostgreSQL
             |                    |
             |                    +-> system configuration, audit records, credentials
             +-> DNS providers, SMTP, payment gateways, AI services
```

The Android package is built from the same local React source as the web UI. It does not fetch a remote web application for rendering. The package only supplies platform lifecycle, secure session storage, API transport, and update-link handling.

## Data Ownership

- `User` owns profile, custom name, avatar URL, role, group, ban state, and user-linked business records.
- `UserCredential` owns normalized email, Argon2id hash metadata, verification state, activation state, and session version.
- `AuthToken` owns one-time token hashes and expiry. Raw tokens never persist.
- `AuthSecurityEvent` records privacy-preserving authentication events and keyed IP hashes.
- `DatabaseBackup` and `DatabaseRestoreJob` record server-generated archive metadata and destructive restore state.
- `SystemConfig` owns dynamic settings including registration policy, SMTP, Android version policy, and communication-key hash.

## Session Boundary

Browser sessions use signed JWT cookies. Native sessions use signed JWT bearer tokens with numeric user IDs, session version, native-client marker, and communication-key hash. Middleware checks the credential session version on every request, so password changes and session invalidation revoke older tokens.

Authorization remains server-side. A client key does not grant administrator access, resource ownership, or permission bypass.

## Authentication Cutover

The running application performs additive schema migration only. The destructive v1-to-v2 migration is a console-only command guarded by a verified export, SMTP test, public URL, normalized email validation, duplicate detection, default group check, and explicit confirmation. It creates activation-required credentials without changing user IDs, then removes retired identity columns/configuration only within the successful transaction.

## Maintenance Boundary

Database export and restore never execute browser-provided commands. The service constructs fixed `pg_dump` and `pg_restore` arguments from deployment configuration, stores archive files in a server-only directory, validates ZIP content and recorded checksums before extraction, creates a pre-restore backup, and keeps API traffic blocked after any destructive restore attempt until restart and operator review.
