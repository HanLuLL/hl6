# Client adaptation specification

## Architecture

The client is a Capacitor container for the built `web/` application. It must preserve the same UI, routes, and accessibility behavior as the web product. Do not add duplicate domain, authentication, DNS, credit, or persistence logic to `client/`; use the existing API through `web/src/lib/api.ts`.

## Communication key

- Administrators generate a 256-bit communication key in Admin Settings.
- The server stores only a SHA-256 hash and returns the plaintext only in the generation response.
- Client builds receive the key through `VITE_CLIENT_COMMUNICATION_KEY` and send it as `X-HL6-Client-Key` on API fetches. The server validates every presented key; `/api/v1/client/version` requires one.
- Treat the key as an application credential. It never replaces the OIDC session or authorization checks. Rotate it for every compromised or retired build.

## Version management

- `/api/v1/client/version` returns the latest version, forced-update flag, notice, and update URL after key validation.
- The embedded web app checks this endpoint when `VITE_CLIENT_BUILD_VERSION` is present. A forced update cannot be dismissed.
- Update links must use HTTPS. Version values use `0-9`, letters, `.`, `+`, and `-`, with a maximum of 32 characters.

## OIDC

The client uses the same OIDC web callback: `https://your-hl6-domain.example/api/v1/auth/callback`. Register that HTTPS callback and the site homepage as logout redirect URI with the provider. Do not create a native OIDC client unless a provider specifically requires it; the packaged web UI keeps the callback, session-cookie, and logout behavior consistent.

## Change record

| Date | Change |
| --- | --- |
| 2026-07-14 | Added Capacitor shell, client communication-key API, version gate, Android version metadata, build workflow, and OIDC adaptation rules. |

Every client-related code or workflow change must update this document in the same change set.
