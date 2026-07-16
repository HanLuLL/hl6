# Deployment and Upgrade

## Prerequisites

- Docker Engine 24+ and Docker Compose v2.
- PostgreSQL 16, supplied by `docker-compose.prod.yml` unless an external database is used.
- A public HTTPS hostname for the web application and email links.
- A working SMTP account before enabling registration or migrating existing users.

## Images

```bash
# Global registry
ghcr.io/hanlull/hl6:v2.0.0

# Mainland China proxy
ghcr.milu.moe/hanlull/hl6:v2.0.0
```

Set `HL6_IMAGE` to either address. The proxy is only a pull-path alternative; it must not be used as an image build source.

## Environment

Create `.env` from `.env.example` and set production values:

```dotenv
POSTGRES_DB=hl6
POSTGRES_USER=hl6
POSTGRES_PASSWORD=replace-with-a-random-password
APP_URL=https://domain.example.com
FRONTEND_URL=https://domain.example.com
BACKEND_URL=https://domain.example.com
ALLOWED_ORIGINS=https://domain.example.com
AUTH_PASSWORD_PEPPER_ID=v1
AUTH_PASSWORD_PEPPER=replace-with-a-long-random-secret
ENCRYPTION_KEY=optional-64-character-hex-value
SMTP_BOOTSTRAP_HOST=smtp.example.com
SMTP_BOOTSTRAP_PORT=587
SMTP_BOOTSTRAP_USERNAME=mailer@example.com
SMTP_BOOTSTRAP_PASSWORD=replace-with-the-smtp-password
SMTP_BOOTSTRAP_FROM_NAME=HL6
SMTP_BOOTSTRAP_FROM_ADDR=mailer@example.com
SMTP_BOOTSTRAP_USE_TLS=true
SMTP_BOOTSTRAP_ENABLED=true
MAINTENANCE_DATA_DIR=/var/lib/hl6/maintenance
```

`AUTH_PASSWORD_PEPPER` is mandatory before local authentication can be enabled. Do not place it in the repository, release notes, APK inputs, or database export.

`MAINTENANCE_DATA_DIR` is mounted as the `maintenance-data` Docker volume. It stores server-generated backup archives temporarily; copy a production backup to independent encrypted storage immediately after downloading it.

`SMTP_BOOTSTRAP_*` values are copied into database settings only when the corresponding SMTP value is missing. This is the secure first-start path for email verification before any administrator exists. Set `ENCRYPTION_KEY` in production so the copied password is encrypted, then manage SMTP through the administration console after the first administrator signs in.

## Start

```bash
docker compose --env-file .env -f docker-compose.prod.yml pull
docker compose --env-file .env -f docker-compose.prod.yml up -d
docker compose --env-file .env -f docker-compose.prod.yml ps
curl -fsS https://domain.example.com/health
```

The service migrates additive schema on startup. It does not automatically delete legacy identity columns or enable the new authentication system.

## SMTP and First Registration

1. Set working `SMTP_BOOTSTRAP_*` values before the first startup.
2. Register through the public email-verification page and set the first password.
3. The first locally registered user on an empty installation becomes the administrator.
4. After signing in, open **Administration -> Email Notifications** to confirm or change SMTP, then use **Test Send**.
5. Configure the registration policy in **Administration -> Access and Registration**.

## v1 to v2 Upgrade

The hard cutover is required only for installations with existing users from v1.

1. Deploy the v2 image with `AUTH_PASSWORD_PEPPER` and working `SMTP_BOOTSTRAP_*` values. Existing-user databases remain intentionally unavailable for local login before the explicit cutover.
2. Send a verified SMTP test from the container:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin mail test --recipient administrator@example.com
```

3. Find an administrator ID and create the required verified database archive:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec postgres \
  sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT id, email FROM users WHERE role = '\''admin'\'' ORDER BY id;"'

docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin maintenance export --created-by-user-id 1
```

Record the returned backup `id` and retain the ZIP outside the host.

4. Run preflight:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin auth preflight
```

5. Resolve every reported blocker. Do not proceed around duplicate normalized emails or a missing backup.
6. Run the cutover:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin auth cutover --confirm --backup-id 123
```

7. Restart the application:

```bash
docker compose --env-file .env -f docker-compose.prod.yml restart app
```

Existing users now activate a local password through their historic email. Their business data remains linked to the same user ID.

## Rollback

The pre-cutover database archive is the rollback path. Do not start an older binary against a partly cut-over database.

1. Stop the application.
2. Restore the verified pre-cutover archive through **Data Maintenance** or an isolated recovery environment.
3. Validate the restored database and restart the matching application image.

## Reverse Proxy

Proxy HTTPS to the application port. Preserve `Host`, `X-Forwarded-Proto`, `X-Forwarded-Host`, `Origin`, `Authorization`, `X-HL6-Client-Key`, and `X-Idempotency-Key`. Do not cache API mutations, email-link pages, or server-sent events.

The server accepts Capacitor's `https://localhost` origin for the packaged Android client. `ALLOWED_ORIGINS` is for additional browser origins; the reverse proxy must still preserve the Android request headers listed above.
