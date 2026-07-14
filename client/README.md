# HL6 client shell

The Android client packages the same `web/` React application through Capacitor. It has no duplicate domain logic: all data access remains in the HL6 API and the rendered UI is the web UI.

Build variables:

- `VITE_API_BASE_URL`: `https://your-domain.example/api/v1`
- `VITE_CLIENT_COMMUNICATION_KEY`: generated in Admin Settings and returned only once
- `VITE_CLIENT_BUILD_VERSION`: version checked by the in-app update guard
- `CLIENT_APP_NAME` and `CLIENT_APP_ID`: Capacitor app metadata

The build workflow also writes `VITE_CLIENT_BUILD_VERSION` to Android versionName and a deterministic numeric versionCode.

Use the repository workflow for reproducible Android builds. The communication key is compiled into the client bundle, so it is an application credential, not a substitute for user authentication. Rotate it immediately if an APK is exposed outside the intended release channel.
