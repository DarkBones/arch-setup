#!/usr/bin/env bash
set -euo pipefail

# --- configurable (or override via env) ---
AUR_PKG="${AUR_PKG:-bas-tui}"
AUR_DIR="${AUR_DIR:-$HOME/tmp/$AUR_PKG}"
AUR_SSH="${AUR_SSH:-ssh://aur@aur.archlinux.org/${AUR_PKG}.git}"
BUMP_MODE="${BUMP_MODE:-patch}"
# ------------------------------------------

usage() {
    cat <<EOF
Usage:
  $0 --version X.Y.Z
  $0 [--bump major|minor|patch]
  $0 --stamp

Env overrides:
  AUR_PKG, AUR_DIR, AUR_SSH, BUMP_MODE
EOF
}

# Parse args
VERSION=""
STAMP=0
while [[ $# -gt 0 ]]; do
    case "$1" in
    --version)
        VERSION="${2:?}"
        shift 2
        ;;
    --bump)
        BUMP_MODE="${2:?major|minor|patch}"
        shift 2
        ;;
    --stamp)
        STAMP=1
        shift
        ;;
    -h | --help)
        usage
        exit 0
        ;;
    *)
        echo "Unknown arg: $1" >&2
        usage
        exit 2
        ;;
    esac
done

SRC_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$SRC_ROOT" ]] || {
    echo "Run from inside your source git repo." >&2
    exit 1
}
cd "$SRC_ROOT"

# Ensure clean tree
if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "Working tree not clean. Commit or stash first." >&2
    exit 1
fi

# Canonicalize GitHub URL from 'origin'
REMOTE="$(git remote get-url --push origin)"
GH_HTTPS="$(echo "$REMOTE" |
    sed -E \
        -e 's#^git@github\.com:(.+)/(.+)\.git$#https://github.com/\1/\2#' \
        -e 's#^ssh://git@github\.com/(.+)/(.+)\.git$#https://github.com/\1/\2#' \
        -e 's#^https://github\.com/(.+)/(.+)(\.git)?$#https://github.com/\1/\2#')"
GH_USER="$(echo "$GH_HTTPS" | sed -E 's#https://github.com/([^/]+)/.*#\1#')"
GH_REPO="$(echo "$GH_HTTPS" | sed -E 's#https://github.com/[^/]+/([^/]+).*#\1#')"

# Determine VERSION
if [[ $STAMP -eq 1 && -z "$VERSION" ]]; then
    VERSION="$(date +%Y.%m.%d.%H%M)"
fi
if [[ -z "$VERSION" ]]; then
    git fetch --tags --quiet || true
    LAST="$(git tag -l 'v*' | sort -V | tail -n1 | sed 's/^v//')"
    LAST="${LAST:-0.0.0}"
    IFS=. read -r MA MAJ MI MIN PA PAT <<<"${LAST//./ }"
    MA=${MA:-0}
    MI=${MI:-0}
    PA=${PA:-0}
    case "$BUMP_MODE" in
    major) VERSION="$((MA + 1)).0.0" ;;
    minor) VERSION="${MA}.$((MI + 1)).0" ;;
    patch | *) VERSION="${MA}.${MI}.$((PA + 1))" ;;
    esac
fi
TAG="v${VERSION}"

# Create & push tag
echo "==> Tagging ${TAG}"
git tag "$TAG"
git push origin "$TAG"

# Prepare (or update) local AUR repo checkout
if [[ -d "$AUR_DIR/.git" ]]; then
    echo "==> Updating local AUR repo at $AUR_DIR"
    git -C "$AUR_DIR" fetch origin
    git -C "$AUR_DIR" reset --hard origin/master
else
    echo "==> Cloning AUR repo to $AUR_DIR"
    git clone "$AUR_SSH" "$AUR_DIR"
fi

# Update PKGBUILD: pkgver, pkgrel, url (to keep it in sync), checksums, .SRCINFO
echo "==> Bumping AUR PKGBUILD to ${VERSION}-1"
cd "$AUR_DIR"

# Ensure we have a PKGBUILD
[[ -f PKGBUILD ]] || {
    echo "PKGBUILD missing in $AUR_DIR"
    exit 1
}

# Bump fields
sed -i -E "s|^pkgver=.*$|pkgver=${VERSION}|" PKGBUILD
sed -i -E "s|^pkgrel=.*$|pkgrel=1|" PKGBUILD
sed -i -E "s|^url=.*$|url=\"${GH_HTTPS}\"|" PKGBUILD

# Refresh checksums (needs pacman-contrib)
if command -v updpkgsums >/dev/null 2>&1; then
    updpkgsums
else
    echo "WARN: updpkgsums not found; leaving sha256sums as-is."
fi

# Regenerate .SRCINFO
makepkg --printsrcinfo >.SRCINFO

# Commit & push
git config user.name "${GIT_AUTHOR_NAME:-DarkBones}"
git config user.email "${GIT_AUTHOR_EMAIL:-${GH_USER}@users.noreply.github.com}"
git add PKGBUILD .SRCINFO
git commit -m "${AUR_PKG} ${VERSION}-1"
git push origin HEAD:master

echo "==> Done. Install with: yay -S ${AUR_PKG}"
