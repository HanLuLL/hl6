# Development

## Local Stack

```bash
make db-up
make dev-server
make dev-web
```

The web development server proxies `/api` to the Go server. Root `.env` is read by Vite through `envDir: ".."`.

## Required Local Variables

```dotenv
DATABASE_URL=postgres://hl6:hl6dev@localhost:5433/hl6?sslmode=disable
AUTH_PASSWORD_PEPPER_ID=dev-v1
AUTH_PASSWORD_PEPPER=development-only-long-random-value
FRONTEND_URL=http://localhost:5174
BACKEND_URL=http://localhost:8081
ALLOWED_ORIGINS=http://localhost:5174,https://localhost
```

Do not use production password peppers, SMTP credentials, payment secrets, DNS keys, signing keys, or communication keys in local development.

## Checks

```bash
cd web
pnpm run lint
pnpm run build

cd ../server
go test ./internal/auth ./internal/migration ./internal/service ./internal/repository ./internal/middleware -count=1
go build ./cmd/server
go build ./cmd/hl6-admin
```

Handler integration tests requiring PostgreSQL use `HL6_TEST_DATABASE_URL`. Use an isolated database. The maintenance restore test must never target a shared or production database.

## Android

```bash
cd web
pnpm run build
pnpm exec cap sync android
cd android
./gradlew :app:assembleDebug --no-daemon
```

Review [agent.md](agent.md) before any UI, auth, API, client version, or packaging change.
