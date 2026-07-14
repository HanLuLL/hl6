# HL6 Native Android Client

This directory is a pure native Android application written in Kotlin and Jetpack Compose. It contains no WebView, browser renderer, HTML bundle, JavaScript runtime, or H5 fallback.

## Scope

- Render native UI only, using tokens derived from `web/src/index.css`.
- Store the client communication key and user access token with Android Keystore-backed encrypted preferences.
- Call HL6 APIs and render their returned state.
- Enforce a server-provided version policy at startup. The server, not the APK, compares versions.
- Launch OIDC in the system browser through Custom Tabs, then exchange a one-time native code through the API.

The client does not make DNS, credit, permission, validation, or business decisions locally. Those decisions remain on the HL6 server.

## Build configuration

`client/scripts/configure-build.mjs` writes the ignored `client/local.properties` file from environment variables. It requires:

- `CLIENT_COMMUNICATION_DOMAIN`
- `CLIENT_COMMUNICATION_KEY`
- `CLIENT_DISPLAY_NAME`
- `CLIENT_VERSION_NAME`
- `CLIENT_APPLICATION_ID`
- `CLIENT_KEYSTORE_FILE`
- `CLIENT_KEYSTORE_PASSWORD`
- `CLIENT_KEY_ALIAS`
- `CLIENT_KEY_PASSWORD`

The GitHub workflow supplies the first five through dispatch inputs and the signing values from repository secrets. The resulting APK is a signed release APK.

## OIDC

The native redirect URI is generated as `hl6.<applicationId>://auth/callback`. The app first calls `/auth/native/start` with `X-HL6-Client-Key`; the server records the approved redirect URI and returns a short-lived browser URL. The app opens only that URL in Custom Tabs. After the provider returns to the server, the server redirects to the deep link with a short-lived, one-time code. The app exchanges that code with `X-HL6-Client-Key` for a Bearer session token, and the server requires the key again for every protected native-session request.

The bundled communication key is necessarily present in the APK build artifact. Android Keystore protects its persisted copy, but an application-wide key is not a replacement for user authentication. Rotate or revoke it from HL6 Admin Settings whenever a build is compromised.
