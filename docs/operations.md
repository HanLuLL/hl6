# Operations

## Health and Logs

```bash
curl -fsS https://domain.example.com/health
docker compose --env-file .env -f docker-compose.prod.yml logs -f app
```

Logs may contain request path, status, and sanitized errors. They must not contain password values, raw email tokens, communication keys, session tokens, SMTP secrets, database URLs with passwords, payment secrets, or private DNS credentials.

## Authentication Incidents

| Symptom | Action |
| --- | --- |
| Registration email is not sent | Verify SMTP in Administration, send a test email, inspect email log, and verify public frontend URL confirmation. |
| Existing user cannot activate | Check normalized email collisions with `hl6-admin auth preflight`; use console recovery only when mail delivery cannot be restored. |
| All users are signed out | Verify pepper/session configuration and database health. Password reset or session-version changes intentionally invalidate old sessions. |
| Password hashes fail after rotation | Restore the prior pepper as `AUTH_PREVIOUS_PASSWORD_PEPPER` and retain it until successful sign-ins rehash accounts. |

## Database Export and Restore

Export archives are short-retention operational files, not your only disaster recovery copy. Download them, verify the recorded SHA-256, encrypt them independently, and test restoration in an isolated environment.

During a restore, maintenance mode blocks API access except restore-job status. If `pg_restore` cannot start at all, such as when the executable is unavailable, the failed job releases maintenance mode. Once `pg_restore` has started, maintenance mode remains active whether the command succeeds or fails. Restart only after an operator has validated the restored database or recovered from the retained pre-restore archive; never reopen normal traffic against a partially restored database.

## Cutover Recovery

Before v2 cutover, retain the verified archive outside the server. If cutover fails before completion, resolve the preflight error and retry only after validating database state. If a completed cutover must be abandoned, restore the verified pre-cutover archive and use a matching application image; never point an older image at a partially migrated database.

For a v1 database, create the required verified pre-cutover archive with the v2 container command before running `auth preflight`:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin maintenance export --created-by-user-id <administrator-id>
```

The command is intentionally console-only, validates the operator is an administrator, and records the backup ID used by the later cutover.

If the migration uses SMTP bootstrap configuration, verify delivery and record the required preflight timestamp before the cutover:

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin mail test --recipient administrator@example.com
```

## Android Operations

Build a new APK before rotating the communication key. Verify direct APK URL, SHA-256, app package, signed certificate, and version policy before enabling forced update. Keep the previous optional-update path available until the new APK is confirmed installable.
