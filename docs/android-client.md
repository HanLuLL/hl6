# Android Client

## Delivery Model

Android packages the repository's locally built `web/` React application with Capacitor. It does not load a remote site as its application UI and it does not contain a second copy of business rules. Page rendering, routes, styles, icons, translations, and interaction components are shared with the web build.

The server remains authoritative for authentication, authorization, profile updates, domains, DNS, credits, bans, appeals, validation, data maintenance, and update policy.

## Native Authentication

The Android login, registration, activation, and password-reset pages call the same email authentication APIs as the web application. A successful Android login receives a native bearer session only when the request includes the valid `X-HL6-Client-Key`. The session is persisted with the Android secure-storage plugin backed by Android Keystore.

Every protected native request sends both:

```text
Authorization: Bearer <session>
X-HL6-Client-Key: <build communication key>
```

The client key is generated and revoked in the web administration console. It is intentionally treated as a build identifier, not a user credential.

The packaged build supplies the identifier on first launch, then the client persists the active value with Android Keystore-backed storage and replaces it when a newer build carries a different value. Because a distributed APK can always be inspected, this identifier is not an unextractable secret and must never be relied on for user authorization, privilege, or ownership decisions.

## UI Compatibility

Any change to web routes, UI, API payloads, fields, errors, session behavior, profile behavior, ban notices, update flow, or translations must be adapted to Android in the same change. See [agent.md](agent.md).

## Build Workflow

Use `.github/workflows/client-build.yml` with these inputs:

| Input | Requirement |
| --- | --- |
| `communication_domain` | HTTPS backend domain without path, query, fragment, or credentials |
| `communication_key` | 32-byte URL-safe key generated in HL6 administration |
| `client_name` | Android display name |
| `client_icon` | Optional HTTPS PNG/WebP URL or repository-relative PNG/WebP path |
| `version` | `major.minor.patch` versionName |
| `android_package_name` | Lowercase Android application ID |

Configure these GitHub Actions secrets before dispatching a signed build:

```text
ANDROID_KEYSTORE_BASE64
ANDROID_KEYSTORE_PASSWORD
ANDROID_KEY_ALIAS
ANDROID_KEY_PASSWORD
```

The workflow validates inputs, masks sensitive values, builds the local UI, synchronizes it into Android, dynamically injects name/icon/package/version, signs the release APK, verifies its signature, and publishes direct `.apk` artifacts with SHA-256 metadata. It never archives the APK in a ZIP.

## Formal Release

Use `.github/workflows/release.yml` only after the Android workflow has published the matching APK. The formal-release workflow accepts the repository-controlled GitHub Pages paths only:

```text
https://<owner>.github.io/<repository>/android/<application-id>/latest.apk
https://<owner>.github.io/<repository>/android/<application-id>/manifest.json
```

It rejects redirects, arbitrary HTTPS hosts, query strings, mismatched package/version metadata, mismatched checksums, unexpected signing certificates, and a manifest that does not identify the release commit. The published GitHub Release attaches the raw APK and its `.sha256` file directly.

## Updates

At startup the client requests:

```text
GET /api/v1/client/version?current_version=<version>
```

The web administration console decides latest version, force-update flag, notice, and HTTPS update link. A normal update can be dismissed. A forced update blocks entry until the user follows the supplied link.

When an older build presents a revoked or invalid communication key, `/client/version` exposes only a forced-update recovery response. Normal API calls remain rejected. Administrators must configure a valid newer version and HTTPS update URL before rotating or revoking an active key.

## CORS

The server accepts Capacitor's `https://localhost` origin for the packaged Android client. Keep these request headers available through the reverse proxy:

```text
Origin, Content-Type, Accept, Authorization, X-HL6-Client-Key, X-Idempotency-Key
```
