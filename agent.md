# Client Adaptation Entry Point

All work affecting the packaged Android client must follow [docs/agent.md](docs/agent.md).

The Android package is built from the local shared React UI and communicates with HL6 through APIs. It must not load a remote website as its primary UI, duplicate server business logic, or bypass server authorization.

For v2, native authentication uses direct email/password APIs, Android Keystore-backed session storage, and the server-managed communication-key header. Any client-affecting change must update `docs/agent.md` in the same change set.
