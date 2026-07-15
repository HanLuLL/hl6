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

`CLIENT_ICON_PATH` is optional. It accepts a repository-relative PNG/WebP file or a direct HTTPS PNG/WebP URL. Remote images are fetched without credentials, cannot redirect, are limited to 2 MiB, and are validated by file signature before being written into Android resources. A supplied icon is generated as an ignored `client_icon_custom` resource; the tracked default icon is never modified.

## GitHub Actions release signing

Before the first release build, configure these repository-level Actions secrets in GitHub under **Settings > Secrets and variables > Actions**:

- `ANDROID_KEYSTORE_BASE64`: one-line Base64 representation of the release keystore.
- `ANDROID_KEYSTORE_PASSWORD`: password protecting that keystore.
- `ANDROID_KEY_ALIAS`: alias of the signing key.
- `ANDROID_KEY_PASSWORD`: signing key password. Use the same value as the keystore password for a PKCS12 keystore.

Create the keystore once and keep it for every later release. Replacing it prevents Android from installing a newer APK over an existing installation. For example, run the following locally with Java `keytool`, then use the prompted passwords as the corresponding GitHub secrets:

```bash
keytool -genkeypair -keystore hl6-release.keystore -storetype PKCS12 -alias hl6-release -keyalg RSA -keysize 4096 -validity 9125
base64 -w 0 hl6-release.keystore
```

On Windows PowerShell, create the Base64 value with:

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes(".\hl6-release.keystore"))
```

Never commit the keystore, its Base64 value, or either password. The workflow lists missing configuration names without printing secret values and validates the keystore before Gradle starts.

For a keystore containing exactly one private-key entry, the workflow detects its internal alias and keystore type through `keytool` before Gradle starts. This supports PKCS12 files exported by Windows certificate tools as well as JKS files generated with `keytool`; no signing value is printed in the build log.

## OIDC

The native redirect URI is generated as `hl6.<applicationId>://auth/callback`. The app first calls `/auth/native/start` with `X-HL6-Client-Key`; the server records the approved redirect URI and returns a short-lived browser URL. The app opens only that URL in Custom Tabs. After the provider returns to the server, the server redirects to the deep link with a short-lived, one-time code. The app exchanges that code with `X-HL6-Client-Key` for a Bearer session token, and the server requires the key again for every protected native-session request.

The bundled communication key is necessarily present in the APK build artifact. Android Keystore protects its persisted copy, but an application-wide key is not a replacement for user authentication. Rotate or revoke it from HL6 Admin Settings whenever a build is compromised.
