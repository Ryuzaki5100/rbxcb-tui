# Releasing rbxcb-tui

This document covers the active `.deb` release pipeline and the optional
publishing channels for AUR, Homebrew, Fedora COPR, Snap, and a future signed
apt repository.

Package name everywhere: **`rbxcb-tui`**  
License: **MIT**  
Maintainer: `Ryuzaki5100 <Ryuzaki5100@users.noreply.github.com>`  
First release: **`v0.1.0`**

## Using the update-project skill

The repository includes an agent skill at
`.opencode/skills/update-project/SKILL.md`.

When invoked, it:

1. Inspects the latest tag, `VERSION`, and current GitHub Release.
2. Asks whether to keep or bump the version.
3. For **keep same version**, rebuilds artifacts and replaces the existing
   GitHub Release assets with matching checksums.
4. For **new version**, updates version references, commits, tags, pushes, and
   verifies the release workflow.
5. Offers optional AUR, Homebrew, COPR, Snap, and apt-channel publishing only
   when the corresponding repository variable and secret exist.

Same-version replacements change checksums. Snap cannot publish the same version
twice; AUR should increment `pkgrel`; RPM should increment `Release`. Avoid
replacing a published apt package for the same upstream version.

## 1. Active release: GitHub Release + `.deb`

The active workflow is `.github/workflows/release.yml`.

It runs on:

- tags matching `v*`
- manual `workflow_dispatch` with an existing tag

It runs tests on Ubuntu, macOS, and Windows, then uses GoReleaser to build:

- Linux `amd64` / `arm64` archives
- macOS `amd64` / `arm64` archives
- Windows `amd64` zip
- checksums
- `.deb` package for Linux amd64/arm64
- GitHub Release assets

No repository key or extra secret is required beyond `GITHUB_TOKEN`.

### Cut a release

```bash
git tag -a v0.1.0 -m "rbxcb-tui v0.1.0"
git push origin v0.1.0
```

Verify the release contains:

```text
rbxcb-tui_0.1.0_linux_amd64.deb
rbxcb-tui_0.1.0_linux_arm64.deb
rbxcb-tui_0.1.0_linux_amd64.tar.gz
rbxcb-tui_0.1.0_linux_arm64.tar.gz
rbxcb-tui_0.1.0_darwin_amd64.tar.gz
rbxcb-tui_0.1.0_darwin_arm64.tar.gz
rbxcb-tui_0.1.0_windows_amd64.zip
checksums.txt
```

### Test the `.deb`

```bash
dpkg-deb -I rbxcb-tui_0.1.0_linux_amd64.deb
dpkg-deb -c rbxcb-tui_0.1.0_linux_amd64.deb
lintian rbxcb-tui_0.1.0_linux_amd64.deb
sudo apt install ./rbxcb-tui_0.1.0_linux_amd64.deb
rbxcb-tui --version
```

## 2. Disabled/manual channels

The following workflows exist but are gated behind a manual dispatch flag and a
repository variable:

| Channel | Workflow | Repository variable | Secrets |
|---|---|---|---|
| AUR | `publish-aur.yml` | `ENABLE_AUR` | `AUR_SSH_PRIVATE_KEY` |
| Homebrew | `publish-homebrew.yml` | `ENABLE_HOMEBREW` | `HOMEBREW_TAP_TOKEN` |
| Fedora COPR | `publish-copr.yml` | `ENABLE_COPR` | `COPR_LOGIN`, `COPR_API_TOKEN` |
| Snap | `publish-snap.yml` | `ENABLE_SNAP` | `SNAPCRAFT_STORE_CREDENTIALS` |
| Signed apt repo | `publish-apt-repo.yml` | `ENABLE_APT_REPO` | `GPG_PRIVATE_KEY`, `GPG_PASSPHRASE` |

To enable one:

1. Set up the account/repository described below.
2. Add the secret(s) in GitHub repository settings.
3. Add the repository variable with value `true`.
4. Run the workflow manually:
   - provide the release tag,
   - set `publish=true`.

Nothing is published while the corresponding variable is unset/false.

## 3. AUR

### One-time setup

1. Create an account at <https://aur.archlinux.org>.
2. Generate an SSH key:

   ```bash
   ssh-keygen -t ed25519 -C "aur-rbxcb-tui" -f ~/.ssh/aur
   cat ~/.ssh/aur.pub
   ```

3. Add the public key to your AUR account.
4. Test:

   ```bash
   ssh -T aur@aur.archlinux.org
   ```

5. Submit a new package named `rbxcb-tui` in the AUR web interface.
6. Clone it:

   ```bash
   git clone ssh://aur@aur.archlinux.org/rbxcb-tui.git
   ```

7. Add `AUR_SSH_PRIVATE_KEY` to GitHub secrets. Use the private key file
   contents.
8. Set `ENABLE_AUR=true`.

### Template

`packaging/aur/PKGBUILD` uses placeholders:

- `__VERSION__`
- `__SHA256__`

The workflow downloads the tagged source tarball, computes the checksum,
substitutes the placeholders, generates `.SRCINFO`, and publishes.

### Manual publish

```bash
cd rbxcb-tui
cp ../rbxcb-tui/packaging/aur/PKGBUILD .
# replace placeholders, then:
makepkg --printsrcinfo > .SRCINFO
makepkg -si
namcap PKGBUILD
git add PKGBUILD .SRCINFO
git commit -m "rbxcb-tui 0.1.0"
git push
```

Official Arch `extra` requires a separate package review; this project targets
the AUR.

## 4. NixOS and nixpkgs

### Flake usage

```bash
nix run github:Ryuzaki5100/rbxcb-tui
nix profile install github:Ryuzaki5100/rbxcb-tui
```

### Official nixpkgs

Official nixpkgs publishing is a manual PR. No keys are required, only a GitHub
account.

1. Fork `NixOS/nixpkgs`.
2. Add `pkgs/by-name/rb/rbxcb-tui/package.nix` based on
   `packaging/nix/package.nix`.
3. Replace both `lib.fakeHash` placeholders:
   - source hash
   - `vendorHash`
4. Add an entry for yourself under `maintainers/maintainer-list.nix` if needed.
5. Build:

   ```bash
   nix-build -A rbxcb-tui
   ```

6. Run:

   ```bash
   nixpkgs-review pr <number>
   ```

7. Open a PR and respond to maintainer review.

`packaging/nix/package.nix` is a template only; it is not referenced by the
flake.

## 5. Homebrew

### One-time setup

1. Create a public repository:

   ```text
   Ryuzaki5100/homebrew-tap
   ```

2. Add a `Formula/` directory.
3. Create a fine-grained GitHub token with contents write access to the tap.
4. Add `HOMEBREW_TAP_TOKEN` to GitHub secrets.
5. Set `ENABLE_HOMEBREW=true`.

### Publishing

`publish-homebrew.yml` fills `packaging/homebrew/rbxcb-tui.rb.tmpl` with the
Darwin archive checksums and pushes `Formula/rbxcb-tui.rb` to the tap.

Users install with:

```bash
brew tap Ryuzaki5100/tap
brew install rbxcb-tui
```

### Validation

```bash
brew audit --new-formula rbxcb-tui
brew install --build-from-source ./Formula/rbxcb-tui.rb
brew test rbxcb-tui
```

Official Homebrew Core requires a separate PR and notability review. The tap is
the supported path for now.

### Optional macOS notarization

Required secrets:

```text
APPLE_CERT_P12
APPLE_CERT_PASSWORD
APPLE_ID
APPLE_TEAM_ID
APPLE_APP_PASSWORD
```

When configured, the release pipeline should:

1. Import the Developer ID Application certificate.
2. Codesign the Darwin binary.
3. Submit with `xcrun notarytool`.
4. Staple the ticket.
5. Build archives from the signed/notarized binary.

If the secrets are absent, unsigned binaries are published and Gatekeeper may
require manual approval.

## 6. Fedora COPR

### One-time setup

1. Create a Fedora account.
2. Sign in to <https://copr.fedorainfracloud.org>.
3. Create project `ryuzaki5100/rbxcb-tui`.
4. Generate a COPR API token.
5. Add secrets:
   - `COPR_LOGIN`
   - `COPR_API_TOKEN`
6. Set `ENABLE_COPR=true`.

### Publishing

`publish-copr.yml` runs in a Fedora container, builds an SRPM from
`packaging/rpm/rbxcb-tui.spec`, and triggers COPR.

Users install with:

```bash
sudo dnf copr enable ryuzaki5100/rbxcb-tui
sudo dnf install rbxcb-tui
```

Official Fedora repositories require a package review and are not automated.

## 7. Snap

### One-time setup

1. Create an Ubuntu One account.
2. Register the snap name:

   ```bash
   snapcraft register rbxcb-tui
   ```

3. Build and test locally:

   ```bash
   snapcraft
   sudo snap install --classic --dangerous ./rbxcb-tui_*.snap
   ```

4. Export store credentials:

   ```bash
   snapcraft export-login \
     --snaps=rbxcb-tui \
     --acls=package_push \
     --channels=stable \
     snap-login.txt
   ```

5. Store the file contents as `SNAPCRAFT_STORE_CREDENTIALS`.
6. Set `ENABLE_SNAP=true`.

Classic confinement may require review by the Snap Store.

Users install with:

```bash
sudo snap install rbxcb-tui --classic
```

## 8. Signed apt repository (future)

The `.deb` release works today. A signed apt repository is optional.

### One-time setup

1. Generate a GPG key:

   ```bash
   gpg --full-generate-key
   ```

2. Export the private key:

   ```bash
   gpg --armor --export-secret-keys <key-id>
   ```

3. Add secrets:
   - `GPG_PRIVATE_KEY`
   - `GPG_PASSPHRASE`
4. Set `ENABLE_APT_REPO=true`.
5. Complete `packaging/apt/publish.sh` for your chosen hosting layout.

The disabled workflow is a template; it intentionally exits until the repository
layout and signing/publishing steps are finalized.

## 9. Secrets and variables checklist

Repository secrets:

```text
AUR_SSH_PRIVATE_KEY
HOMEBREW_TAP_TOKEN
COPR_LOGIN
COPR_API_TOKEN
SNAPCRAFT_STORE_CREDENTIALS
GPG_PRIVATE_KEY
GPG_PASSPHRASE
APPLE_CERT_P12             # optional
APPLE_CERT_PASSWORD        # optional
APPLE_ID                   # optional
APPLE_TEAM_ID              # optional
APPLE_APP_PASSWORD         # optional
```

Repository variables:

```text
ENABLE_AUR
ENABLE_HOMEBREW
ENABLE_COPR
ENABLE_SNAP
ENABLE_APT_REPO
```

## 10. Windows

Windows currently receives a `.exe` archive in GitHub Releases. Scoop, winget,
and code signing are intentionally not configured yet.
