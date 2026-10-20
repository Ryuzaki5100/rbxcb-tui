---
name: update-project
description: Update the GitHub Releases page and release assets. Use when asked to create or update a release, keep or bump the version, refresh release notes, rebuild release artifacts, or publish packages.
---

# Update Project Releases

Use this skill when the user asks to release the project, update the GitHub
Releases page, rebuild release assets, bump the version, or publish package
manager channels.

## Rules

- Always ask whether to keep the current version or publish a new version before
  making any release changes.
- Never create, rotate, or delete credentials.
- Never enable package-manager channels unless the user explicitly asks.
- Do not push commits or tags until the user confirms the version decision.
- Keep the existing release-note style.
- Remove any stale references to the previous version when bumping.

## Step 1: Inspect the repository

Run:

```bash
git status
git branch --show-current
git describe --tags --abbrev=0
cat VERSION
gh release view <tag> --json tagName,name,body,assets,isDraft,isPrerelease
git log --oneline <previous-tag>..HEAD
```

Read:

- `docs/RELEASING.md`
- `.github/workflows/release.yml`
- `.goreleaser.yaml`
- `packaging/`

If no tag exists, propose `v0.1.0`.

## Step 2: Ask the version question

Always ask:

> Keep the current version `<vX.Y.Z>`, or publish an updated version?

Offer:

- Keep current version
- Patch
- Minor
- Major
- Custom

Do not continue until the user answers.

## Step 3A: Keep the current version

This path rebuilds and replaces the existing GitHub Release assets.

1. Check out the existing tag:

   ```bash
   git fetch --tags
   git checkout <tag>
   ```

2. Validate:

   ```bash
   go vet ./...
   go test ./...
   go test -race ./internal/...
   E2E_SCRAMBLES=20 go test ./e2e -run E2E
   nix build
   goreleaser check
   ```

3. Rebuild assets without publishing:

   ```bash
   goreleaser release --clean --skip=publish
   ```

4. Replace matching GitHub Release assets:

   ```bash
   gh release upload <tag> \
     dist/*.tar.gz \
     dist/*.zip \
     dist/*.deb \
     dist/checksums.txt \
     --clobber
   ```

5. Regenerate release notes and update the release:

   ```bash
   gh release edit <tag> --notes-file /tmp/release-notes.md
   ```

6. Warn the user that:

   - checksums changed,
   - package-manager caches may retain old hashes,
   - Snap cannot publish the same version twice,
   - COPR/RPM should use an incremented package release,
   - apt repositories should not replace a published `.deb` for the same
     upstream version.

7. Ask whether to update optional channels.

Same-version channel rules:

| Channel | Same-version behavior |
|---|---|
| GitHub Release | Replace assets with `--clobber` |
| Homebrew tap | Update formula checksums; warn about cached bottles |
| AUR | Increment `pkgrel`, update `sha256sums` |
| COPR / RPM | Increment RPM `Release` |
| Snap | Requires a new upstream version |
| apt repo | Avoid replacing a published `.deb` |

## Step 3B: Publish a new version

1. Compute the new semantic version.

2. Update:

   - `VERSION`
   - README installation examples
   - `docs/RELEASING.md` examples
   - `packaging/nix/package.nix`
   - any hardcoded release examples

3. Generate release notes grouped by:

   - Features
   - Fixes
   - Packaging and distribution
   - Documentation
   - Tests

4. Validate:

   ```bash
   go vet ./...
   go test ./...
   go test -race ./internal/...
   E2E_SCRAMBLES=20 go test ./e2e -run E2E
   nix build
   goreleaser check
   goreleaser release --snapshot --clean --skip=publish
   ```

5. Commit:

   ```bash
   git add VERSION README.md docs/RELEASING.md packaging/nix/package.nix
   git commit -m "release: vX.Y.Z"
   ```

6. Tag and push:

   ```bash
   git tag -a vX.Y.Z -m "rbxcb-tui vX.Y.Z"
   git push origin main
   git push origin vX.Y.Z
   ```

7. Watch the release workflow:

   ```bash
   gh run list --workflow release.yml
   gh run watch <run-id>
   ```

8. Verify:

   ```bash
   gh release view vX.Y.Z
   ```

Confirm archives, `.deb`, checksums, and notes are present.

## Step 4: Optional channels

Only run these if the user confirms and the required repository variable and
secret exist.

```bash
gh workflow run publish-aur.yml -f tag=vX.Y.Z -f publish=true
gh workflow run publish-homebrew.yml -f tag=vX.Y.Z -f publish=true
gh workflow run publish-copr.yml -f tag=vX.Y.Z -f publish=true
gh workflow run publish-snap.yml -f tag=vX.Y.Z -f publish=true
gh workflow run publish-apt-repo.yml -f tag=vX.Y.Z -f publish=true
```

Current variables:

- `ENABLE_AUR`
- `ENABLE_HOMEBREW`
- `ENABLE_COPR`
- `ENABLE_SNAP`
- `ENABLE_APT_REPO`

Never create accounts, register package names, or enable variables
automatically.

## Step 5: Final report

Summarize:

- version decision,
- commit and tag pushed,
- release URL,
- assets rebuilt or reuploaded,
- checksum changes,
- optional channels published or skipped,
- validation results,
- remaining manual setup steps.
