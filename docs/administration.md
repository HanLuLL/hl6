# Administration

## Access and Registration

This area controls new registration only:

- Enable or disable registration.
- Select unrestricted registration, exact-domain allowlist, or exact-domain blocklist.
- Manage normalized domain entries.
- Check whether local authentication is active after production cutover.

Invalid wildcard entries are rejected. Existing users can still activate or reset a password when registration is disabled.

## Site and Appearance

This area owns public frontend/backend addresses, announcement, footer, and SEO fields. Environment-supplied URLs are read-only in the UI. Confirm the current external URL before sending authentication links.

## Email Notifications

Configure SMTP host, port, credentials, sender name, sender address, TLS, and enabled state. Secrets are masked on read and encrypted when an `ENCRYPTION_KEY` is configured. A successful test send records the SMTP test time required by the authentication cutover preflight.

Ban notifications include the ban reason, start time, and expected unban time. A permanent ban is represented explicitly rather than an invented date.

## Payment and Integrations

Payment gateway URL, merchant identifier, merchant secret, and per-channel switches belong here. Existing DNS provider accounts, content review models, and external service credentials retain their dedicated administration pages.

## Android Client

The Android section controls:

- Latest semantic version.
- Optional or force-update behavior.
- Update announcement and HTTPS update URL.
- Communication key generation and revocation.

The communication key is shown only once when generated. The server stores only its SHA-256 hash. A key inside an APK identifies a configured client build; it never grants a user role or bypasses ownership, ban, or permission checks.

Before rotating or revoking an existing key, configure a newer semantic version and a valid HTTPS update URL. The server enables forced update recovery, so a client with an invalidated key can retrieve only the replacement-update response and cannot use normal APIs. Generate, install, and verify the new APK before revoking the old key.

## Data Maintenance

Only administrators can export or overwrite the database.

### Export

**Export Database** runs a fixed `pg_dump` command and downloads a ZIP containing:

- `database.dump` in PostgreSQL custom format.
- `manifest.json` with version and source metadata.
- `SHA256SUMS` for the dump and manifest.

No browser-provided command, SQL, host, path, or database URI is accepted.

### Restore

Restore requires all of the following in one administrator session:

1. A ZIP generated in the accepted archive format.
2. The current password.
3. A server-issued, short-lived, single-use restore challenge.
4. The exact confirmation phrase `RESTORE DATABASE`.

The server validates ZIP paths, file count, sizes, manifest, checksums, and custom-dump header before extracting. It creates a mandatory pre-restore backup, enters maintenance mode, runs fixed `pg_restore` arguments, validates health and required tables, records the restore job, and keeps the API in maintenance mode until the application is restarted.

Do not close the maintenance incident until the post-restart health check succeeds.
