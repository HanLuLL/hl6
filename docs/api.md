# API Guide

Base path: `/api/v1`.

Responses use this envelope:

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

Errors use a nonzero `code`, HTTP status, human-readable `message`, and optional stable `message_key`.

## Authentication

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/auth/registration/request` | Request a verification email for a new registration |
| POST | `/auth/activation/request` | Request an activation email for an existing migrated user |
| POST | `/auth/password/forgot` | Request a password reset email without account enumeration |
| POST | `/auth/password/complete` | Complete registration, activation, or reset with one-time token and password |
| POST | `/auth/login` | Email/password sign-in |
| GET | `/auth/me` | Current user and credit balance |
| POST | `/auth/logout` | Clear the current session |
| PUT | `/auth/profile` | Update name, avatar URL, bio, or website |

### Request Examples

```json
POST /auth/login
{
  "email": "user@example.com",
  "password": "a password of at least twelve characters"
}
```

```json
POST /auth/password/complete
{
  "token": "one-time-url-safe-token",
  "password": "a password of at least twelve characters"
}
```

Browser sessions are cookies. Android sign-in adds `X-HL6-Client-Key` and receives `access_token` in the response data. Protected native requests add `Authorization: Bearer <access_token>` and the same client-key header. Logging out increments the credential session version and invalidates issued browser and Android JWTs.

## Ban State

When a signed-in user is banned, protected requests return `403` with `reason`, `banned_at`, and `banned_until`. The application allows the signed-in user to resolve `/auth/me`, ban information, appeals, and logout so the ban screen can display complete timing and submit an appeal.

## Android Version

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/client/version?current_version=2.0.0` | Latest version, update flag, force-update flag, announcement, and URL |

Valid clients send `X-HL6-Client-Key`. A missing or invalid key receives only a forced-update recovery payload when an administrator has configured replacement metadata; it never permits normal API access.

## Administration

All administration routes require an authenticated administrator.

| Method | Path | Purpose |
| --- | --- | --- |
| GET/PUT | `/admin/settings/access` | Registration and exact-domain policy |
| GET | `/admin/security-events` | Authentication security event list |
| GET/PUT | `/admin/config` | Existing site, mail, SEO, and integration configuration surface |
| POST | `/admin/config/url-confirm` | Confirm current public URL configuration |
| GET/PUT | `/admin/client/config` | Android version and update policy |
| POST | `/admin/client/communication-key` | Generate a communication key, returned once |
| DELETE | `/admin/client/communication-key` | Revoke active communication key |
| POST | `/admin/maintenance/export` | Generate and download server-controlled database ZIP |
| POST | `/admin/maintenance/restore/challenge` | Issue short-lived restore challenge after password verification |
| POST | `/admin/maintenance/restore` | Restore a validated archive after fresh password/challenge/confirmation |
| GET | `/admin/maintenance/restores` | List database restore jobs and maintenance state |

### Access Settings

```json
{
  "registration_enabled": true,
  "domain_policy_mode": "allowlist",
  "domain_policy_domains": ["example.com", "example.org"]
}
```

Allowed modes are `unrestricted`, `allowlist`, and `blocklist`. Domains are exact matches after lowercasing and IDNA normalization.

### Database Restore

The restore request is multipart form data with these fields:

```text
archive=<validated ZIP>
password=<current administrator password>
challenge=<one-time server challenge>
confirmation=RESTORE DATABASE
```

The server rejects arbitrary dump formats, paths, SQL, and connection information. Successful restore responses include `restart_required: true` and maintenance remains active until restart.

## Idempotency and Origin

Mutating API calls should provide `X-Idempotency-Key`. Cookie-backed browser mutations require a trusted `Origin`; native bearer sessions require a valid client key. Do not proxy-strip `Authorization`, `Origin`, `X-HL6-Client-Key`, or `X-Idempotency-Key`.
