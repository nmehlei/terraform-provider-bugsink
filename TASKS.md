# Kickoff checklist

- [x] Confirm repository owner, public provider namespace, and license. (owner/namespace: `nmehlei`, revisit if Bugsink wants to adopt it — see docs/architecture.md; license: Apache-2.0)
- [ ] File `docs/upstream-proposal.md` with Bugsink.
- [ ] Wait for and record upstream lifecycle and token decisions.
- [x] Initialize Go module (`src/bugsink` client package; provider scaffold still pending, see below).
- [x] Add disposable Bugsink Docker Compose test environment.
- [ ] Implement and test team/project create/read/update, with Delete failing
      with an actionable diagnostic instead of removing state silently (see
      AGENTS.md's "Delete-endpoint policy", amended 2026-08-24 — this no
      longer waits on upstream delete support).
- [ ] Add import, generated docs, GoReleaser, signed release workflow, and Registry publication.
