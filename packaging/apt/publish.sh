#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GPG_PRIVATE_KEY:-}" || -z "${GPG_PASSPHRASE:-}" ]]; then
  echo "GPG_PRIVATE_KEY and GPG_PASSPHRASE are required" >&2
  exit 1
fi

echo "This is a template for publishing a signed apt repository."
echo "1. Build or download the .deb for the requested tag."
echo "2. Create a repository directory with dists/stable/main/binary-amd64."
echo "3. Generate Packages/Release with apt-ftparchive."
echo "4. Sign Release with the configured GPG key."
echo "5. Publish the directory to a gh-pages/apt branch."
echo
echo "Implement the repository setup here before enabling ENABLE_APT_REPO."
exit 1
