---
name: publish-terraform-provider
description: Use when an nmehlei Terraform provider (a GitHub repo nmehlei/terraform-provider-<name> with GoReleaser releases) must be published to, or checked on, the public Terraform Registry — e.g. "connect the registry", "publish the provider", or `terraform init` reports "provider registry … does not have a provider named". Checks the release's GPG signature against the namespace keys first, fixes a key mismatch by re-releasing, then publishes through HCP Terraform in a Playwright browser the user has signed into. Never signs in, never reads or handles private keys.
---

# Publish a provider to the Terraform Registry

The `nmehlei` namespace is managed in **HCP Terraform** (organization `nmehlei` → Registry → Public
namespaces). The legacy GitHub sign-in on registry.terraform.io is deprecated; use HCP Terraform.

## Hard stops

1. **Never sign in.** The user signs in to HCP Terraform in the Playwright browser (username/password
   form at `https://app.terraform.io/login`). On a login page, stop and ask.
2. **Never read, export or paste a private key or passphrase.** Key material moves only by the user
   running a command themselves (`! …` in the prompt).
3. **Never delete a GitHub release that is already on the Registry**, and never delete one without
   the user's OK. Deleting a release that is not yet on the Registry is fine once they agree.
4. Publishing accepts HCP Terraform's Terms of Use; say so in the report.

## 1. Is it already there?

```sh
curl -s -o /dev/null -w '%{http_code}\n' https://registry.terraform.io/v1/providers/nmehlei/<name>/versions
```

200 → already published; check the versions and stop. 404 → continue.

## 2. Does the release's signing key belong to the namespace?

The Registry rejects a release signed with a key it does not know. Compare the release's signature
issuer with the namespace keys (public data only):

```sh
gh release download <tag> -R nmehlei/terraform-provider-<name> -p '*SHA256SUMS.sig' -O release.sig
gpg --list-packets release.sig | grep -oE 'keyid [0-9A-F]+'          # the signing (sub)key
curl -s 'https://registry.terraform.io/v2/gpg-keys?filter%5Bnamespace%5D=nmehlei' \
  | python3 -c 'import sys,json;[open(k["attributes"]["key-id"]+".asc","w").write(k["attributes"]["ascii-armor"]) for k in json.load(sys.stdin)["data"]]'
for f in *.asc; do gpg --show-keys --keyid-format long "$f" | grep -E '^(pub|sub)'; done
```

A signature usually names a **subkey**: it matches if the key id is a `pub` or `sub` of a
registered key. As of 2026-09-30 the namespace has `6301595BFED94920` (signing subkey
`AF7200341C2FAF14`; Bugsink, Aptabase, GlitchTip) and `2C6BCA6EE6AFA291`.

Match → step 4. No match → step 3.

## 3. Re-sign with a registered key

Preferred over registering another key: one key for every provider.

1. The user sets the repository secret from their keyring (they run it, not you):

   ```
   ! gpg --armor --export-secret-keys 6301595BFED94920 | gh secret set GPG_PRIVATE_KEY -R nmehlei/terraform-provider-<name>
   ```

   If gpg prints `waiting for lock (held by <pid>)` and then `WARNING: nothing exported`, the
   secret has just been **overwritten with nothing**. Check that `<pid>` no longer exists
   (`ps -p <pid>`), remove `~/.gnupg/public-keys.d/pubring.db.lock` and the matching `.#lk…<pid>`
   file, run `gpgconf --kill keyboxd`, and have the user run the command again. Confirm with
   `gh api repos/nmehlei/terraform-provider-<name>/actions/secrets` that `GPG_PRIVATE_KEY` was
   updated seconds ago (timestamps only).
2. Check that GitVersion will compute the same version for the same commit once the tag is gone
   (clone to a scratch dir, `git tag -d <tag>`, run `gitversion /showvariable SemVer`).
3. With the user's OK: `gh release delete <tag> -R … --cleanup-tag --yes`, then
   `gh workflow run release.yml -R … --ref main` and `gh run watch <id> --exit-status`.
4. Verify the new signature against the namespace key, not just the run's success:

   ```sh
   mkdir -m 700 vk && gpg --homedir vk --import 6301595BFED94920.asc
   gpg --homedir vk --verify *_SHA256SUMS.sig *_SHA256SUMS   # must say "Good signature"
   ```

## 4. Publish in HCP Terraform

1. Playwright: `https://app.terraform.io/app/registry/public-namespaces` → the user signs in →
   organization `nmehlei`.
2. **Publish → Provider**. The list shows only repositories named `terraform-provider-<NAME>`
   that the HCP GitHub App can see; if one is missing, the page links to the app's permissions.
3. Select the repository, pick a category (e.g. *Logging & Monitoring* for error trackers), tick
   the Terms of Use, **Publish provider**. All existing releases with a semantic version are
   imported; later releases publish automatically.
4. Poll until the Registry lists the version:

   ```sh
   curl -s https://registry.terraform.io/v1/providers/nmehlei/<name>/versions
   ```

5. Delete the Playwright snapshots written during the run (`.playwright-mcp/`): they contain pages
   of the signed-in session.

## 5. Consumers

In a repository that uses the provider, refresh the lock file for every platform:

```sh
terraform -chdir=<dir> providers lock -platform=linux_amd64 -platform=darwin_amd64 -platform=darwin_arm64
```
