# Kickoff checklist

- [x] Confirm repository owner, public provider namespace, and license. (owner/namespace: `nmehlei`, revisit if Bugsink wants to adopt it — see docs/architecture.md; license: Apache-2.0)
- [x] File `docs/upstream-proposal.md` with Bugsink — filed as
      [bugsink/bugsink#495](https://github.com/bugsink/bugsink/issues/495)
      (2026-08-26).
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
- [x] Registry publication: live at
      https://registry.terraform.io/providers/nmehlei/bugsink. Required
      creating an HCP Terraform organization (`nmehlei`, Personal type —
      HCP Terraform now requires an org container even for solo publishing),
      connecting GitHub via the HCP Terraform GitHub App, and adding the
      release-signing GPG public key (`.secrets-local/gpg-public-key.asc`,
      not committed) to the namespace's Signing Keys before "Publish
      provider" would accept the repo. `terraform init` with
      `source = "nmehlei/bugsink"` now works with no special config.
