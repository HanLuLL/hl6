# HL6 Documentation

## Start Here

1. [Deployment](deployment.md): production environment, Docker images, reverse proxy, SMTP, v1 cutover, and rollback.
2. [Authentication](authentication.md): registration, activation, password reset, domain policies, sessions, and password-pepper rotation.
3. [Administration](administration.md): grouped administration areas, Android controls, export, and restore.
4. [Android Client](android-client.md): local UI package model, communication key, signing secrets, and workflow inputs.

## Reference

| Need | Document |
| --- | --- |
| Endpoints, payloads, and error behavior | [API](api.md) |
| Components, data ownership, and security boundaries | [Architecture](architecture.md) |
| Monitoring, backup recovery, and incident response | [Operations](operations.md) |
| Local development and test commands | [Development](development.md) |
| Mandatory Android adaptation process | [Agent Contract](agent.md) |

## Version Scope

These documents describe `v2.0.0`. The authentication model is HL6-owned email/password authentication; provider login endpoints and callback configuration are not part of this release.
