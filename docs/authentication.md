# Email Authentication

## Overview

HL6 v2 uses first-party email/password authentication. Passwords are validated by the server, hashed with Argon2id plus a server-side pepper, and never returned, logged, emailed, or stored in reversible form.

The web application uses secure, HttpOnly, SameSite=Lax session cookies. The packaged Android client receives a short-lived bearer session only after presenting its communication key and stores that session with Android Keystore-backed secure storage.

## New Registration

1. Open `/register` and submit an email address.
2. HL6 evaluates registration availability and the administrator's exact-domain policy, then sends a one-time verification link without exposing whether an account already exists.
3. The verification link opens `/set-password`.
4. Set a password between 12 and 128 Unicode code points.
5. HL6 atomically creates the user, local credential, default group, credit balance, referral state, and session.

The raw verification token is URL-safe, random, short-lived, single-use, stored only as a SHA-256 hash, and removed from the browser address bar immediately after the page reads it.

## Existing v1 Accounts

After the hard cutover, an existing user opens `/activate-account` or `/login`, submits the historic email address, follows the activation link, and sets a new password. The linked `users.id` does not change. Existing profile fields, custom avatar, credit balance, domains, DNS records, bans, appeals, notifications, payments, and audit links remain intact.

No old session is accepted after cutover. Each existing user must establish a local password once.

## Password Recovery

`/forgot-password` always returns a neutral response. Activated accounts receive a one-time reset link. Completing a reset updates the Argon2id hash and increments the credential session version, invalidating other sessions.

Password setup consumes its one-time link before starting the expensive hash derivation. A successful password change also consumes every other outstanding activation, reset, and restore link for that user. If two different links are consumed concurrently, the credential lock rejects the older sibling after the first password change, so it cannot overwrite the new password. Logging out increments the same session version, so copied browser or Android bearer tokens cannot remain usable after logout.

Authentication requests are independently rate-limited by normalized email address and privacy-preserving client IP hash. Changing either value does not bypass the other limit.

## Email Domain Policy

Administrators configure this at **Administration -> Access and Registration**:

- `unrestricted`: any syntactically valid address may register.
- `allowlist`: only exact normalized domains in the list may register.
- `blocklist`: valid addresses except exact normalized domains in the list may register.

Wildcard domains, partial suffixes, malformed IDNs, duplicate entries, and ambiguous domain spelling are rejected. This policy applies to new registration only. Existing users remain able to activate and reset their password.

## Password Pepper

Set these deployment secrets outside PostgreSQL:

```dotenv
AUTH_PASSWORD_PEPPER_ID=v1
AUTH_PASSWORD_PEPPER=a-high-entropy-random-secret
AUTH_PREVIOUS_PASSWORD_PEPPER_ID=
AUTH_PREVIOUS_PASSWORD_PEPPER=
```

To rotate, set a new current ID/value and place the former pair in the previous variables. A successful login using the previous pepper transparently rehashes the password with the current pepper. Remove the previous pair only after users have had sufficient time to sign in.

## v1 Cutover

The cutover is deliberately console-only. It will not run from the browser or Android client.

1. Deploy the v2 image with `AUTH_PASSWORD_PEPPER`, a public `FRONTEND_URL`, and the `SMTP_BOOTSTRAP_*` values required for a working sender. The bootstrap values populate only missing SMTP database settings. Existing-user databases remain intentionally disabled for local login until the explicit switch completes.
2. Send and verify an SMTP test from the v2 container. This records the timestamp required by preflight without needing an OIDC session:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin mail test --recipient administrator@example.com
```

3. Identify an existing administrator ID from PostgreSQL, then create and retain the server-recorded export through the v2 console command:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec postgres \
  sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT id, email FROM users WHERE role = '\''admin'\'' ORDER BY id;"'

docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin maintenance export --created-by-user-id 1
```

The export command prints the verified backup record. Copy its `id`, download or retain the archive from the maintenance volume, and keep an independent encrypted copy before continuing.

4. Run the preflight command:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app /app/hl6-admin auth preflight
```

5. Resolve every blocker, especially blank/invalid emails, normalized duplicate emails, missing default group, SMTP test status, public HTTPS frontend URL, and verified backup.
6. Execute the irreversible switch with the exported backup ID:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin auth cutover --confirm --backup-id 123
```

7. Restart the application. Existing users then use the activation flow.

For mail-delivery recovery, a deployment operator can create a new one-time activation link without writing it to the database or logs:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin auth issue-activation --email user@example.com
```

Deliver that output only through a secure channel.

## QQ Avatar Fallback

If a user has no custom avatar and their normalized address matches numeric `@qq.com`, clients render:

```text
https://q.qlogo.cn/headimg_dl?dst_uin=<QQ_NUMBER>&spec=640&img_type=jpg
```

The URL is HTTPS to avoid mixed-content failures. A custom `avatar_url` always wins, and sign-in never overwrites it.
