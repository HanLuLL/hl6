# Android Client Adaptation Contract

This document is mandatory for every change that affects Android-visible behavior. A server, API, web UI, route, field, error, state, authentication, version, or release change is incomplete until this contract is checked and the Android adaptation is included in the same change set.

## 1. Delivery Boundary

1. Android packages the repository's local `web/` production build through Capacitor.
2. Android must not load a remote website as its main UI source.
3. Android must not duplicate business rules, permission decisions, DNS behavior, payment logic, data validation, or administration decisions.
4. The server remains authoritative; Android renders state returned by APIs and stores only the native session and build configuration needed for transport.

## 2. UI Parity

1. Web and Android use the same React routes, components, styles, design tokens, icons, translations, validation copy, dialogs, loading states, and error states.
2. Web UI changes must be checked on Android viewport sizes in the same iteration.
3. No Android-only reduced layout, alternate component system, stale text, or divergent interaction flow may be introduced.
4. Custom user names and avatars remain server-owned and must not be overwritten by sign-in or client initialization.
5. Ban views and ban email must show ban start time and expected unban time or explicit permanent-ban state.

## 3. API and Session Compatibility

1. Any endpoint, payload, field, validation rule, status code, permission rule, or error-key change must update the Android API client and UI in the same change.
2. Native sign-in uses email/password endpoints directly. Registration, activation, reset, logout, ban state, profile, and update gate must work in the packaged client.
3. Native sessions are stored only with Android Keystore-backed secure storage.
4. Every native request carries `X-HL6-Client-Key`; protected native requests also carry a valid bearer session.
5. The communication key is a server-managed build identifier, not a user authorization substitute.
6. The raw key must never enter Git, application logs, URLs, database records, release notes, or screenshots.

## 4. Communication Key Lifecycle

1. Only the web administration console generates, rotates, and revokes a communication key.
2. The server stores only a SHA-256 hash and returns raw key material once at generation time.
3. Build a and verify a new APK before revoking the previous key.
4. Server authorization must still verify user session, role, ownership, ban state, and resource permission.

## 5. Build and Release

`.github/workflows/client-build.yml` is the Android build workflow. It must validate and dynamically inject:

- Backend HTTPS domain.
- Communication key.
- Client display name.
- Client icon.
- Semantic version.
- Android package name.

The workflow must validate signing secrets, build the local UI, synchronize Capacitor, sign the release APK, verify it with `apksigner`, and publish a direct APK plus SHA-256. Do not wrap the output APK in a ZIP.

The web administration console controls latest version, forced-update flag, update notice, and HTTPS update URL. A forced update may only be enabled after a verified installable APK is available.

## 6. Required Change Checklist

- [ ] Shared web UI and Android viewport behavior match.
- [ ] API contracts, TypeScript types, error handling, and client storage are updated.
- [ ] Login, registration, activation, reset, profile, ban view, and logout are checked when authentication changes.
- [ ] Communication-key handling and Android headers remain intact.
- [ ] Version gate, update notice, force-update behavior, and direct APK link are checked when release behavior changes.
- [ ] Android build workflow inputs, signing validation, and documentation are updated when packaging changes.
- [ ] This document's change record is updated for a client-visible architecture, protocol, or release-rule change.

## 7. Change Record

| Date | Change |
| --- | --- |
| 2026-07-16 | Replaced provider login handoff with direct email/password authentication, secure native bearer sessions, activation, registration, and recovery pages. |
| 2026-07-16 | Removed login deep-link build metadata; retained local shared UI, communication-key transport, Keystore session storage, direct APK workflow, and update gate. |
| 2026-07-16 | Added Android-visible access-policy, client-version, communication-key, and data-maintenance administration surfaces. |
| 2026-07-16 | Removed static Android application-ID and version defaults. Capacitor and Gradle now fail unless the workflow injects build configuration. |
| 2026-07-16 | Persisted the active communication-key build identifier with Android Keystore-backed storage and replace it on upgraded client builds. |
| 2026-07-16 | Hardened password-link concurrency, independent email/IP throttling, restore-gate startup handling, and formal Android release validation. Android releases now accept only the repository-controlled GitHub Pages `latest.apk` and `manifest.json` paths without redirects. |
