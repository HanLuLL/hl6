# HL6

HL6 is a domain, subdomain, and DNS management platform. It provides user-owned subdomain claims, DNS records, credits, content governance, administration, and an Android package built from the same local React UI as the web application.

## v2.0.0

- First-party email authentication with verified registration, password setup, existing-account activation, password reset, Argon2id hashes, session invalidation, and exact email-domain allowlist/blocklist rules.
- Existing v1 users keep the same user IDs, profiles, avatars, credits, domains, DNS records, bans, notifications, and other linked data. They activate a new password through their existing email address after the one-time cutover.
- User profile updates are owned by HL6. A custom name or avatar is never replaced during sign-in. Numeric `@qq.com` addresses receive an HTTPS QQ avatar fallback only when no custom avatar is configured.
- Ban mail and the ban screen show the ban start time and expected unban time.
- The Android client packages the locally built web UI, calls HL6 APIs directly, stores native sessions in Android Keystore-backed storage, and requires the server-managed communication key on native requests.
- Administrators control Android version, force-update policy, update notice, update URL, communication-key lifecycle, and guarded PostgreSQL ZIP export/restore.

## Images

```bash
# Global GHCR image
docker pull ghcr.io/hanlull/hl6:v2.0.0

# Mainland China proxy
docker pull ghcr.milu.moe/hanlull/hl6:v2.0.0
```

Use `latest` for the most recent stable image. The proxy address mirrors the same image content and is useful where direct GHCR access is unavailable.

## Quick Start

```bash
git clone https://github.com/HanLuLL/hl6.git
cd hl6
cp .env.example .env
```

Set at least these production values in `.env`:

```dotenv
POSTGRES_DB=hl6
POSTGRES_USER=hl6
POSTGRES_PASSWORD=replace-with-a-strong-random-password
APP_URL=https://domain.example.com
AUTH_PASSWORD_PEPPER_ID=v1
AUTH_PASSWORD_PEPPER=replace-with-a-long-random-secret
SMTP_BOOTSTRAP_HOST=smtp.example.com
SMTP_BOOTSTRAP_USERNAME=mailer@example.com
SMTP_BOOTSTRAP_PASSWORD=replace-with-the-smtp-password
SMTP_BOOTSTRAP_FROM_ADDR=mailer@example.com
SMTP_BOOTSTRAP_ENABLED=true
```

Start the production stack:

```bash
docker compose --env-file .env -f docker-compose.prod.yml up -d
docker compose --env-file .env -f docker-compose.prod.yml ps
```

For a new installation, SMTP bootstrap values seed only missing server settings so the first email/password registration can complete before an administrator exists. After that, manage SMTP in the administration console. Before migrating an existing v1 installation, follow the cutover procedure in [Deployment](docs/deployment.md).

## Documentation

| Topic | Document |
| --- | --- |
| Documentation index | [docs/README.md](docs/README.md) |
| Email authentication and v1 migration | [docs/authentication.md](docs/authentication.md) |
| Deployment, upgrade, backup, and rollback | [docs/deployment.md](docs/deployment.md) |
| Administration console | [docs/administration.md](docs/administration.md) |
| Android client and GitHub Actions build | [docs/android-client.md](docs/android-client.md) |
| API integration | [docs/api.md](docs/api.md) |
| Architecture | [docs/architecture.md](docs/architecture.md) |
| Operations and recovery | [docs/operations.md](docs/operations.md) |
| Development | [docs/development.md](docs/development.md) |
| Android compatibility contract | [docs/agent.md](docs/agent.md) |

## Development

```bash
make dev
make dev-server
make dev-web
make db-up
make db-down
```

Client-affecting changes must follow [docs/agent.md](docs/agent.md) in the same change set.
