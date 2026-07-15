# Android APK Direct Delivery Design

## Goal

Retain the latest successful Android client build as a directly downloadable `.apk`
without creating a GitHub Release and without exposing an Actions ZIP download to
end users.

## Decision

Publish the signed APK through GitHub Pages from the existing manual Android build
workflow. GitHub's Pages deployment artifact is an internal transport detail with a
one-day retention period; users receive the deployed APK as a normal HTTPS file.

The public download paths for a build are:

- `/android/<application-id>/<application-id>-<version>.apk`
- `/android/<application-id>/latest.apk`
- `/android/<application-id>/manifest.json`

For the current client, the stable update URL will therefore be:

`https://hanlull.github.io/hl6/android/cc.lxii.domain/latest.apk`

## Build And Publish Flow

1. Keep the current web UI build, Capacitor synchronization, release signing, and
   `apksigner` verification steps unchanged.
2. After signature verification, copy the APK into a temporary Pages directory.
3. Generate `manifest.json` with the application ID, display name, version, commit,
   build time, versioned APK path, stable APK path, and SHA-256 digest. The manifest
   must never contain the communication key or signing data.
4. Upload only the temporary Pages directory with
   `actions/upload-pages-artifact` and retain the transport artifact for one day.
5. A separate deployment job uses `actions/deploy-pages` with only `pages: write`
   and `id-token: write` permissions.
6. Keep signing-material cleanup under `if: always()` so publication failures do
   not leave generated credentials in the runner workspace.

Each deployment replaces the previous Pages payload. The versioned filename and
`latest.apk` both point to the current build; historical binaries are intentionally
not retained.

## Repository Configuration

GitHub Pages must use **GitHub Actions** as its build and deployment source. This is
a one-time repository setting. No personal access token, release permission, or
additional signing secret is required.

## Failure Handling

- Missing APK or failed signature verification stops publication.
- Invalid application IDs or versions continue to fail during existing input
  validation before credentials are decoded.
- Manifest generation and APK copying use strict shell error handling.
- Concurrent Pages deployments are serialized so an older run cannot overwrite a
  newer completed build.
- The job summary reports the direct APK URL, SHA-256 digest, and Pages deployment
  URL without printing sensitive configuration.

## Verification

1. Validate the workflow YAML and inspect the final diff for secrets.
2. Trigger the workflow from `main` with version `1.0.0` and the approved client
   configuration.
3. Confirm signing and Pages deployment jobs succeed.
4. Request the stable URL and require HTTP 200 with a non-empty APK response.
5. Download the file locally, verify its SHA-256 digest against `manifest.json`,
   and verify the APK signature.
6. Confirm no GitHub Release was created.

## Alternatives Rejected

- `actions/upload-artifact`: GitHub always presents the download as a ZIP archive.
- GitHub Release assets: provide direct APK downloads but create a Release, which
  is explicitly out of scope.
- Committing APKs to `main` or an artifact branch: permanently bloats Git history
  and conflicts with the repository's main-only source workflow.
