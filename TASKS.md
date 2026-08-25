# Kickoff checklist

- [x] Confirm repository owner, public provider namespace, and license. (owner/namespace: `nmehlei`, revisit if Bugsink wants to adopt it — see docs/architecture.md; license: Apache-2.0)
- [ ] File `docs/upstream-proposal.md` with Bugsink.
- [ ] Wait for and record upstream lifecycle and token decisions.
- [x] Initialize Go module (`src/bugsink` client package + `src/provider` provider package).
- [x] Add disposable Bugsink Docker Compose test environment.
- [x] Implement and test team/project create/read/update, with Delete failing
      with an actionable diagnostic instead of removing state silently (see
      AGENTS.md's "Delete-endpoint policy", amended 2026-08-24 — this no
      longer waits on upstream delete support). Verified end-to-end against a
      live instance, including import and drift detection.
- [x] Add import, generated docs, GoReleaser, signed release workflow.
      GitVersion (Mainline) auto-tags and releases every push to `main` —
      see `.github/workflows/release.yml`. `v0.1.0`/`v0.1.1` published with
      signed checksums.
- [ ] Registry publication: the one remaining manual step is connecting
      `nmehlei/terraform-provider-bugsink` at registry.terraform.io (sign in
      with GitHub, "Publish provider", upload the release-signing GPG public
      key from `.secrets-local/gpg-public-key.asc` under Signing Keys). Not
      automatable — requires an interactive login.
