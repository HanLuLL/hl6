# Android APK Direct Delivery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish the latest signed Android client as a directly downloadable APK through GitHub Pages without creating a GitHub Release.

**Architecture:** Extend the existing manual client build with a temporary static Pages payload created only after `apksigner` succeeds. Upload that payload through GitHub's official Pages artifact action, then deploy it in a separate least-privilege job and expose a stable `latest.apk` URL plus a metadata manifest.

**Tech Stack:** GitHub Actions, Bash, Node.js 22, Capacitor Android, GitHub Pages, SHA-256

## Global Constraints

- Keep version `1.0.0`, package `cc.lxii.domain`, and the existing approved runtime build inputs.
- Do not create a GitHub Release or commit APK binaries to Git.
- Do not print or persist the communication key or signing credentials in files, summaries, manifests, or logs.
- Retain only the latest deployed build; do not promise historical APK hosting.
- Do not add unit tests or run a duplicate local application compile; use static checks and the real GitHub build as acceptance verification.
- Record every client-related change in `docs/agent.md` in the same change set.

---

### Task 1: Add Direct APK Pages Publication

**Files:**
- Modify: `.github/workflows/client-build.yml`

**Interfaces:**
- Consumes: the verified APK at `web/android/app/build/outputs/apk/release/app-release.apk` and validated `CLIENT_APPLICATION_ID`, `CLIENT_VERSION_NAME`, and `CLIENT_DISPLAY_NAME` environment variables.
- Produces: `android/<application-id>/latest.apk`, a versioned APK, `manifest.json`, build outputs `apk_relative_path` and `apk_sha256`, and a `github-pages` deployment.

- [ ] **Step 1: Run the publication contract check before implementation**

```powershell
$workflow = Get-Content -Raw .github/workflows/client-build.yml
foreach ($required in @('actions/upload-pages-artifact@v3', 'actions/deploy-pages@v4', 'latest.apk', 'manifest.json')) {
  if (-not $workflow.Contains($required)) { throw "Missing expected Pages publication contract: $required" }
}
```

Expected: FAIL on the first missing Pages publication contract.

- [ ] **Step 2: Add serialized deployment and build outputs**

Add workflow-level serialization:

```yaml
concurrency:
  group: android-client-pages
  cancel-in-progress: false
```

Add these outputs to the existing `build` job:

```yaml
outputs:
  apk_relative_path: ${{ steps.pages_payload.outputs.apk_relative_path }}
  apk_sha256: ${{ steps.pages_payload.outputs.apk_sha256 }}
```

- [ ] **Step 3: Prepare and upload the Pages payload after signature verification**

Add these steps before signing-material cleanup:

```yaml
      - name: Prepare direct APK download
        id: pages_payload
        run: |
          set -euo pipefail
          apk_source="web/android/app/build/outputs/apk/release/app-release.apk"
          pages_root="$RUNNER_TEMP/hl6-pages"
          app_dir="$pages_root/android/$CLIENT_APPLICATION_ID"
          asset_name="$CLIENT_APPLICATION_ID-$CLIENT_VERSION_NAME.apk"
          relative_path="android/$CLIENT_APPLICATION_ID/latest.apk"

          test -s "$apk_source"
          mkdir -p "$app_dir"
          cp "$apk_source" "$app_dir/$asset_name"
          cp "$apk_source" "$app_dir/latest.apk"
          apk_sha256="$(sha256sum "$apk_source" | awk '{print $1}')"
          export APK_ASSET_NAME="$asset_name" APK_SHA256="$apk_sha256" PAGES_APP_DIR="$app_dir"

          node <<'NODE'
          const fs = require("node:fs");
          const path = require("node:path");
          const manifest = {
            schema_version: 1,
            application_id: process.env.CLIENT_APPLICATION_ID,
            display_name: process.env.CLIENT_DISPLAY_NAME,
            version: process.env.CLIENT_VERSION_NAME,
            commit: process.env.GITHUB_SHA,
            built_at: new Date().toISOString(),
            sha256: process.env.APK_SHA256,
            apk: {
              latest: "latest.apk",
              versioned: process.env.APK_ASSET_NAME,
            },
          };
          fs.writeFileSync(path.join(process.env.PAGES_APP_DIR, "manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);
          NODE

          : > "$pages_root/.nojekyll"
          printf 'apk_relative_path=%s\n' "$relative_path" >> "$GITHUB_OUTPUT"
          printf 'apk_sha256=%s\n' "$apk_sha256" >> "$GITHUB_OUTPUT"

      - name: Upload APK Pages payload
        uses: actions/upload-pages-artifact@v3
        with:
          path: ${{ runner.temp }}/hl6-pages
          retention-days: 1
```

- [ ] **Step 4: Deploy with least-privilege Pages permissions**

Add a second job:

```yaml
  deploy:
    needs: build
    runs-on: ubuntu-latest
    permissions:
      pages: write
      id-token: write
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - name: Deploy direct APK download
        id: deployment
        uses: actions/deploy-pages@v4

      - name: Report direct APK download
        env:
          PAGE_URL: ${{ steps.deployment.outputs.page_url }}
          APK_RELATIVE_PATH: ${{ needs.build.outputs.apk_relative_path }}
          APK_SHA256: ${{ needs.build.outputs.apk_sha256 }}
        run: |
          set -euo pipefail
          direct_url="${PAGE_URL%/}/$APK_RELATIVE_PATH"
          {
            echo "## Android APK published"
            echo
            echo "Direct APK: $direct_url"
            echo "SHA-256: \`$APK_SHA256\`"
          } >> "$GITHUB_STEP_SUMMARY"
```

- [ ] **Step 5: Run the static publication contract check**

Run the Step 1 PowerShell command again.

Expected: PASS with no output.

- [ ] **Step 6: Check workflow diff hygiene**

Run: `git diff --check -- .github/workflows/client-build.yml`

Expected: exit code 0 and no output.

### Task 2: Synchronize Client Delivery Documentation

**Files:**
- Modify: `docs/agent.md`
- Modify: `docs/native-client.md`

**Interfaces:**
- Consumes: the stable Pages path and latest-only retention behavior from Task 1.
- Produces: operator instructions and a dated mandatory client-adaptation record.

- [ ] **Step 1: Update the permanent delivery rules**

Replace the rule that forbids all APK publication with a rule that permits only the explicitly approved Pages channel, forbids Releases, and requires the manifest to exclude secrets. Add a `2026-07-15` change-record row for direct APK delivery.

- [ ] **Step 2: Update the Android operations guide**

Document the one-time **Settings > Pages > Source: GitHub Actions** configuration, the stable URL format, latest-only retention, `manifest.json`, and the absence of Releases.

- [ ] **Step 3: Verify documentation contracts**

Run:

```powershell
rg -n "GitHub Pages|latest\.apk|manifest\.json|GitHub Release" docs/agent.md docs/native-client.md
```

Expected: both files describe Pages direct delivery and explicitly state that no GitHub Release is created.

- [ ] **Step 4: Check documentation diff hygiene**

Run: `git diff --check -- docs/agent.md docs/native-client.md`

Expected: exit code 0 and no output.

### Task 3: Publish And Verify Version 1.0.0

**Files:**
- No source files changed.
- Download verification target: `artifacts/cc.lxii.domain-1.0.0.apk` (not committed).

**Interfaces:**
- Consumes: the pushed `main` workflow, existing signing Secrets, and approved build inputs.
- Produces: a successful Pages deployment and a verified direct APK URL.

- [ ] **Step 1: Commit and push the implementation to `main`**

```powershell
git add -- .github/workflows/client-build.yml docs/agent.md docs/native-client.md docs/superpowers/plans/2026-07-15-direct-apk-delivery.md
git commit -m "ci: publish direct android apk"
git push origin main
```

Expected: push succeeds and `main` is synchronized with `origin/main`.

- [ ] **Step 2: Configure the repository Pages source once**

In `Settings > Pages`, select **GitHub Actions** as the Pages build and deployment source. Do not configure a custom domain or a branch source.

- [ ] **Step 3: Trigger the client workflow**

Run **Build Web Android Client** from `main` with the approved domain, communication key, client name, icon, version `1.0.0`, and package `cc.lxii.domain`. Do not expose the key in any response or log excerpt.

- [ ] **Step 4: Verify the GitHub jobs**

Expected: the build job passes input validation, web build, Capacitor sync, APK signing, `apksigner verify`, and Pages upload; the deploy job succeeds and reports the direct URL.

- [ ] **Step 5: Verify manifest and APK over HTTPS**

```powershell
$base = 'https://hanlull.github.io/hl6/android/cc.lxii.domain'
$manifest = Invoke-RestMethod "$base/manifest.json"
Invoke-WebRequest "$base/latest.apk" -OutFile artifacts/cc.lxii.domain-1.0.0.apk
$actual = (Get-FileHash artifacts/cc.lxii.domain-1.0.0.apk -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actual -ne $manifest.sha256) { throw 'Downloaded APK digest does not match manifest.json' }
if ($manifest.version -ne '1.0.0') { throw 'Published manifest has the wrong version' }
```

Expected: download succeeds, digest matches, and the published version is `1.0.0`.

- [ ] **Step 6: Confirm repository and Release state**

Run: `git status --short --branch`

Expected: `main` matches `origin/main`; only the ignored local APK may exist. Confirm the workflow did not create a GitHub Release.
