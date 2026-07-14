# Android Client Adaptation Rules

HL6 Android is a Kotlin + Jetpack Compose pure native client. WebView, embedded pages, HTML bundles, H5 shells, JavaScript runtimes, and hybrid rendering are forbidden.

Every web, server, API, field, UI, state, or documentation change that affects Android use must update the native client and [docs/agent.md](docs/agent.md) in the same change set. Changes without the required client adaptation must not be merged or released.

The client renders UI, stores credentials with Android Keystore protection, sends API requests with `X-HL6-Client-Key`, displays server state, and applies server version policy. It must not implement business rules, permission checks, DNS changes, credit decisions, or server-side validation locally.

The full mandatory UI, API, version, signing, OIDC, security, workflow, and change-record specification is maintained in [docs/agent.md](docs/agent.md).
