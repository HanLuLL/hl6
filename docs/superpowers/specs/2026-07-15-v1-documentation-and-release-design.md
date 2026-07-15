# v1.0.0 Documentation and Release Design

## Goal

Replace the current overlapping documentation set with a clear, durable Chinese
documentation system, then recreate the formal `v1.0.0` GitHub Release from the
current `main` commit with the verified Android `1.0.0` APK and bilingual release
notes.

## Scope

The release describes stable product capabilities only. It excludes incidental bug
fixes, CI repair history, temporary debugging details, and internal implementation
notes. The Android APK remains available both as a direct GitHub Pages download and
as a permanent GitHub Release asset.

## Documentation Information Architecture

`README.md` remains the concise repository entry point. It links to a new
`docs/README.md` index and gives only the platform summary, supported deployment
paths, quick start, and major capabilities.

The final `docs/` tree contains the following maintained documents:

| File | Audience | Responsibility |
| --- | --- | --- |
| `README.md` | All readers | Navigation by role and documentation conventions |
| `deployment.md` | Operators | Docker deployment, images, configuration, reverse proxy, upgrade and rollback |
| `administration.md` | Administrators | Domains, DNS accounts, users, groups, credits, payment, branding, notifications, client version policy |
| `oidc.md` | Operators | OIDC provider setup, web callback, Android native login compatibility and CORS |
| `android-client.md` | Android maintainers | Capacitor packaging, signing, communication-key model, Pages delivery, update policy and troubleshooting |
| `development.md` | Contributors | Local setup, commands, repository conventions and development workflow |
| `architecture.md` | Contributors and reviewers | Components, data ownership, authentication, DNS, audit, payment and notification flows |
| `api.md` | Integrators | API envelope, authentication, endpoint families, idempotency and error handling |
| `operations.md` | Operators | Health checks, logs, backup, monitoring, queue behavior and incident diagnosis |
| `agent.md` | Maintainers | Mandatory Android adaptation and change-record rules |

Old duplicated documents (`container-images.md`, `native-client.md`, and
`troubleshooting.md`) are removed after their durable content is consolidated.
Temporary `docs/superpowers` plans and specifications are removed from the final
documentation tree.

## Formal Release Flow

`.github/workflows/release.yml` becomes a manual-only formal release workflow. It
accepts a semantic `vX.Y.Z` version, a versioned HTTPS APK URL, and a release-notes
path. The workflow validates the version, downloads the APK, verifies the Pages
manifest version and SHA-256 digest, builds and pushes the versioned Docker image,
then creates a GitHub Release from the checked-in bilingual notes and attaches the
APK.

The old `v1.0.0` release and remote tag are deleted only after the documentation and
workflow changes are pushed. A manual release run then recreates `v1.0.0` at the
current `main` commit. The local duplicate `1.0.0` tag is left untouched because it
is not the requested GitHub Release tag.

## Release Notes Contract

The v1.0.0 notes use Chinese and English sections with matching feature groups:

1. Domain and multi-provider DNS management.
2. OIDC authentication, profiles, user groups, bans, appeals and notifications.
3. Credits, referral, payment methods and operational controls.
4. Audit rules, scheduled scanning, AI review and moderation lifecycle.
5. Branding, SEO, friend links, email and administrative configuration.
6. Android client packaging, secure session handling, native OIDC, version policy,
   direct APK delivery and update behavior.
7. Docker images, deployment, health, backup and monitoring.

It deliberately does not list minor patches, individual CI changes, or internal
refactoring commits.

## Security and Failure Handling

- The release workflow never accepts or prints the Android communication key or
  signing credentials.
- APK acceptance requires a successful HTTPS download, a matching manifest version,
  matching SHA-256 digest, and a non-empty APK file.
- Release notes are checked into the repository and referenced by path, preventing
  manual editor drift between Chinese and English content.
- The workflow only runs manually, so ordinary source updates cannot create tags or
  releases.
- Deleting the old release and tag occurs after the implementation is pushed and
  immediately before the validated manual release run.

## Verification

1. Verify Markdown navigation targets and workflow YAML contracts locally.
2. Confirm no obsolete public documentation file remains.
3. Confirm `main` is pushed before replacing the old release and tag.
4. Confirm the new workflow run builds the Docker image, verifies the versioned APK,
   and creates `v1.0.0` with exactly the expected APK asset.
5. Download the attached release APK and compare its SHA-256 digest with the Pages
   manifest.
6. Confirm the published release body contains Chinese and English sections and
   only grouped stable capabilities.
