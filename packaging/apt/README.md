# Signed apt repository (optional)

The default release workflow publishes a standalone `.deb` on GitHub Releases.
This directory is for the optional signed apt repository.

Required GitHub secrets:

- `GPG_PRIVATE_KEY`
- `GPG_PASSPHRASE`

Required repository variable:

- `ENABLE_APT_REPO=true`

The disabled `publish-apt-repo.yml` workflow calls `publish.sh`, which builds a
signed repository on a `gh-pages` branch. Users then add the repository key and
source URL as documented in the release runbook.
