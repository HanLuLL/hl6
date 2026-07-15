# Android Client Adaptation Rules

HL6 Android packages the local `web/` React/Vite production build with Capacitor. It reuses the same UI, routes, translations and API client as the web application; the APK must never load a remote website as its UI source.

Every web, server, API, field, UI, state or documentation change that affects Android use must update the client integration and [docs/agent.md](docs/agent.md) in the same change set. Changes without the required client adaptation must not be merged or released.

The Android layer is limited to application startup, deep links, system-browser OIDC, Android Keystore-backed session storage and API transport. Server-side business rules, authorization, validation, DNS changes and credit decisions remain on the server.

The full UI, API, version, signing, OIDC, CORS and change-record specification is maintained in [docs/agent.md](docs/agent.md).
