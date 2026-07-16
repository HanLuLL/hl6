# HL6 v2.0.0 Email Authentication and Operations Design

## Status

Approved for planning. This document records the agreed v2.0.0 design before implementation.

## Goals

1. Remove OIDC completely and make HL6-owned email/password authentication the only sign-in system.
2. Preserve every existing user's HL6-owned business data while requiring that user to establish a local password.
3. Provide verified-email registration, password recovery, configurable email suffix rules, and a polished Authing-inspired UI.
4. Reorganize administration into coherent functional areas with typed settings APIs.
5. Provide administrator-controlled PostgreSQL export and destructive restore using ZIP packages.
6. Keep Android as the only client target and make its local bundled UI use the same authentication and API behavior as the web app.
7. Publish v2.0.0 with direct Android APK assets after migration and release verification succeed.

## Explicit Decisions

- OIDC is a hard cutover. v2 does not retain OIDC login, callback, discovery, environment variables, provider configuration, native authorization-code exchange, browser login, or old-session compatibility.
- Existing users preserve their `users.id` and all related HL6 data. They activate local credentials through their existing email address and then sign in again with email/password.
- Android remains the only packaged client. Windows, Linux, and macOS artifacts are out of scope for v2.
- The Android app keeps the existing local bundled React UI model. It does not load a remote website as its application UI; it calls HL6 APIs.
- Numeric `@qq.com` addresses receive a QQ official-avatar fallback only when a user has not chosen a custom avatar. The production URL is HTTPS because an HTTPS HL6 site would block the requested HTTP image as mixed content.

## Authentication Architecture

### Data Model

`User` remains the owner of profile, role, group, ban state, referral code, and all existing business relationships. The v2 cutover removes the OIDC-only `external_id` database column after a verified backup, because no future auth code may depend on it.

New models:

| Model | Purpose | Important fields |
| --- | --- | --- |
| `UserCredential` | One local identity per user | `user_id` unique, `email_normalized` unique, `password_hash`, `password_hash_version`, `email_verified_at`, `password_set_at`, `session_version`, `activation_required_at` |
| `AuthToken` | One-time email links | `purpose`, nullable `user_id`, normalized email, `token_hash`, encrypted/minimal payload, expiry, consumed time, creation metadata |
| `AuthSecurityEvent` | Security audit trail | nullable `user_id`, outcome, action, IP hash, user-agent summary, timestamp |
| `DatabaseBackup` | Export metadata | creator, filename, checksum, database/schema version, creation time, expiry |
| `DatabaseRestoreJob` | Controlled destructive restore | creator, challenge, input checksum, state, pre-restore backup ID, validation result, failure detail, timestamps |

`User.Email` remains the display and business-contact value. `UserCredential.EmailNormalized` is the authoritative case-insensitive sign-in key. Creation and migration update both values in a single transaction.

### Passwords, Tokens, and Sessions

- Password hashes use Argon2id with per-password random salt, versioned parameters, and a server-side pepper stored outside the database. The implementation supports a current and previous pepper ID so rotation can rehash on the next successful login.
- Passwords are never logged, reversible, emailed, or returned through an API. Server validation is authoritative and accepts 12-128 Unicode code points.
- Email link tokens use at least 256 bits of cryptographic randomness, URL-safe encoding, a server-side hash only, a short expiry, single use, and atomic consumption after password validation succeeds.
- Browser sessions are `Secure`, `HttpOnly`, `SameSite=Lax` cookies. Android receives a short-lived bearer session through a request authenticated by the Android communication key and stores it with Android Keystore-backed secure storage.
- New session JWTs identify the numeric HL6 user ID and include `session_version`. Password setup, reset, password change, account disable, or explicit global sign-out invalidates old sessions by incrementing that version.
- Cookie-authenticated write requests validate trusted origin/referrer information. Android bearer calls require both the valid session and the global communication-key header.

### Registration and Recovery Flows

#### New registration

1. User opens `/register` and enters an email address.
2. Server normalizes and validates the address, checks registration is enabled, evaluates its suffix policy, applies IP/email rate limits, and replies without exposing account existence.
3. Server sends a `registration_verify` email link. No usable user account is created yet.
4. The link opens `/set-password`; the frontend removes the raw token from the visible URL after reading it and sends it only in a POST request. Auth pages set a no-referrer policy.
5. A successful password submission consumes the token and creates `User`, `UserCredential`, default group assignment, credit balance, referral data, and a session in one transaction.
6. The user lands on `/dashboard`.

#### Existing v1 user activation

1. After v2 hard cutover, an existing user enters their historic email at `/login` or `/activate-account`.
2. The server responds generically and sends an `account_activation` link to a matching, eligible legacy user.
3. The link opens the same password-setup UI. Completing it creates the `UserCredential` linked to the existing `users.id` and signs the user in.
4. No OIDC provider is contacted, no OIDC callback remains, and no v1 session is accepted.

#### Forgot password

1. `/forgot-password` accepts an email and always returns a neutral success response.
2. If an activated account exists, the system sends a `password_reset` link.
3. `/reset-password` validates and consumes the token, updates the Argon2id hash, increments `session_version`, and signs in only the completing session.

### Email Suffix Policy

Administration exposes three registration modes:

- `unrestricted`: any syntactically valid email address may register.
- `allowlist`: only exact normalized domains in the configured list may register.
- `blocklist`: all valid domains except exact normalized domains in the configured list may register.

Domain input is lowercased, IDNA-normalized, de-duplicated, and stored as JSON. The policy affects new registrations only. Existing users may activate, log in, and reset passwords even when their historical suffix is no longer accepted for new registrations.

### Security Controls

- Per-email and per-IP throttling is shared across instances through Redis when configured, with a durable PostgreSQL fallback.
- Login responses avoid account enumeration. Administrative diagnostics remain restricted to administrators and audit logs.
- Repeated login, registration, reset, activation, and token failures write structured security events with privacy-preserving IP hashes.
- SMTP must be enabled and pass a server-side test before hard cutover. The preflight report blocks cutover for missing/duplicate user emails or failed SMTP.
- A console-only `hl6-admin auth issue-activation` recovery command requires direct deployment access and writes an audited, one-time activation token. It prevents administrator lockout when mail delivery is unavailable.

## OIDC Removal and Migration

### Preflight

Before the production cutover, the operator must:

1. Produce and retain a verified database export.
2. Configure and test SMTP from the current administration UI.
3. Run `hl6-admin auth preflight`, which reports blank legacy emails, normalized-email collisions, missing default groups, SMTP status, public URL validity, and pending destructive changes.
4. Resolve every blocking item. The command returns a non-zero exit code while blockers remain.

### Cutover

The explicit `hl6-admin auth cutover --confirm` command performs a transactionally guarded, backup-required migration:

1. Creates authentication and audit tables plus needed indexes.
2. Normalizes each eligible existing email and creates an activation-required credential row without a password hash.
3. Removes OIDC callback/native-code tables and OIDC-only configuration keys.
4. Drops the obsolete `users.external_id` column only after the backup and data preflight are marked successful.
5. Marks all old signed sessions invalid; v2 session subjects are numeric user IDs only.
6. Records a high-severity audit event containing counts and the backup identifier, never credentials or raw tokens.

The application source removes OIDC handlers, routes, discovery code, environment variables, frontend setup/callback screens, Android deep-link login logic, types, translations, documentation, and dependencies used only by OIDC. A v1 binary rollback is possible only by restoring the verified pre-cutover backup; it must not be used against a partially cut-over database.

## QQ Avatar Fallback

The avatar resolver follows this order:

1. A non-empty user-selected `avatar_url` always wins.
2. For `^([0-9]{5,12})@qq\\.com$` after email normalization, use:

   `https://q.qlogo.cn/headimg_dl?dst_uin=<QQ_NUMBER>&spec=640&img_type=jpg`

3. Otherwise render the existing generated initials/avatar fallback.

The resolver exists in one shared frontend helper and is used by all avatar components. For existing users, migration never overwrites a custom avatar. New and activated QQ accounts may persist the fallback URL only after a user explicitly selects it; the default display behavior works without storing a third-party URL in the profile.

## Auth UI

Create a shared `AuthShell` for web and Android with routes:

- `/login`
- `/register`
- `/verify-email`
- `/set-password`
- `/activate-account`
- `/forgot-password`
- `/reset-password`

The UI follows the approved Authing-inspired direction: focused brand signal, restrained panel layout, one primary action, readable field errors, password visibility control, loading/error states, mobile-first layout, and complete translations in the existing supported languages. OIDC setup and callback pages are removed.

## Administration Information Architecture

Replace the single miscellaneous settings surface with typed routes and grouped navigation:

| Area | Responsibilities |
| --- | --- |
| Account and access | registration switch, email suffix policy, password/session policy, activation status |
| Site and appearance | branding, announcement, footer, SEO, friend links |
| Mail and notifications | SMTP, templates, test delivery, mail logs, in-app notifications |
| Domains and DNS | domains, provider accounts, reserved prefixes, length policy |
| Billing and integrations | credits, referrals, payment gateways, external service credentials |
| Content governance | audit rules, AI review, cases, appeals |
| Android client | communication key, version, force update, notice, update URL, build guidance |
| Data maintenance | export, import, restore jobs, maintenance state |
| Security and audit | administrator audit log and authentication security events |

Retired administration-settings links redirect to their new logical areas. OIDC paths do not redirect and are removed. New typed endpoints own validation and encryption; the browser no longer submits an arbitrary `Record<string, string>` system-configuration payload. Legacy configuration values are read and migrated into namespaced keys such as `auth.registration.enabled`, `email.smtp.host`, and `android.client.latest_version` without exposing secret values.

## Database Export and Restore

### Export

- Use `pg_dump` custom format rather than JSON/GORM serialization.
- Stream an administrator-requested ZIP containing `database.dump`, `manifest.json`, and `SHA256SUMS`.
- Manifest records HL6 release, schema revision, PostgreSQL major version, creation time, source database identifier, and checksums.
- The image includes only the necessary PostgreSQL client utilities. No API accepts a command, SQL statement, path, or database URI from a user.

### Import and overwrite

1. An administrator uploads a ZIP subject to compressed/uncompressed size, file-count, path, MIME, manifest, version, and checksum limits.
2. The UI requires fresh password reauthentication and a server-issued restore challenge plus the exact confirmation phrase.
3. The server creates a mandatory pre-restore backup before allowing overwrite.
4. The system enters maintenance mode, blocks writes, pauses workers, and uses fixed `pg_restore` arguments to restore the verified custom dump to the configured database.
5. The server validates required tables, schema revision, core counts, and a database health query. It records success/failure and requires an application restart so restored session/configuration state is loaded safely.
6. A failed restore leaves maintenance mode only after the operator explicitly restores the mandatory pre-restore backup or completes the documented recovery procedure.

All export/import actions are audit logged. Restore is deliberately unavailable to non-administrators and cannot be invoked by Android clients.

## Android Client and Build Pipeline

- Android uses the locally bundled React UI and calls the same auth/API endpoints as the web app. OIDC Browser and deep-link login code is removed.
- Login, registration, activation, reset, ban state, profile, and update gate work in the packaged app.
- User session tokens use Keystore-backed secure storage. The build communication key remains required on every Android request and can be rotated/revoked from the Android client admin area.
- A global build key is not a secret device identity because it is packaged into the APK. Server authorization always also verifies the user's session, role, ownership, and ban state.
- The Android workflow retains dynamic domain, communication key, name, icon, version, and package-name inputs. It validates them, signs the APK, verifies its signature, emits a direct APK, and records SHA-256.
- The release workflow builds/collects the signed v2 APK and attaches direct `.apk` assets plus checksum files to the v2.0.0 GitHub Release. No desktop artifacts are produced.

## Documentation

Update these documents as part of the same change:

- `README.md` and `docs/README.md`
- `docs/deployment.md`
- `docs/administration.md`
- `docs/authentication.md` (new)
- `docs/android-client.md`
- `docs/api.md`
- `docs/architecture.md`
- `docs/operations.md`
- `docs/development.md`
- `docs/agent.md`

Delete `docs/oidc.md` and all OIDC-only deployment/configuration text. `docs/agent.md` gains the Android compatibility rule and a v2 change record before client-affecting code is merged.

## Verification and Release Gates

Before pushing to `main` or publishing v2.0.0, validate:

1. Migration preflight catches duplicate/blank legacy email cases and cutover preserves all user-linked records.
2. Registration, verification, activation, login, password reset, logout, session invalidation, suffix modes, and ban visibility work on web and Android.
3. No OIDC routes, config keys, environment variables, frontend pages, Android login dependency, or documentation remain.
4. QQ avatar fallback renders through HTTPS and never replaces an existing custom avatar.
5. Admin navigation, typed settings APIs, secret masking, and redirects from retired settings URLs work.
6. Export ZIP validates and a controlled overwrite restore works against an isolated PostgreSQL fixture, including mandatory rollback backup behavior.
7. Android build produces a signed APK, starts locally, authenticates with the new API, passes update-gate behavior, and matches the current web UI.
8. Release assets contain the direct Android APK, checksum, bilingual release notes, and the Docker image tags for `v2.0.0` and `latest`.

## Non-Goals

- No Windows, Linux, or macOS client for v2.
- No OIDC compatibility mode, provider migration, or old-session bridge.
- No arbitrary SQL console or unrestricted database file upload.
- No use of the Android communication key as a substitute for user authorization.
