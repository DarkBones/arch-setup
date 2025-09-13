#!/usr/bin/env bash
set -euo pipefail

# --- configurable (or override via env) ---
AUR_PKG="${AUR_PKG:-bas-tui}"
AUR_DIR="${AUR_DIR:-$HOME/tmp/$AUR_PKG}"
AUR_SSH="${AUR_SSH:-ssh://aur@aur.archlinux.org/${AUR_PKG}.git}"
BUMP_MODE="${BUMP_MODE:-patch}"     # major|minor|patch (ignored if --version given)
# ------------------------------------------

usage() {
  cat <<EOF
Usage:
  $0 --version X.Y.Z
  $0 [--bump major|minor|patch]   # default: patch
  $0 --stamp                      # version = YYYY.MM.DD.HHMM

Env overrides: AUR_PKG, AUR_DIR, AUR_SSH, BUMP_MODE
EOF
}

# ----- args -----
VERSION=""
STAMP=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="${2:?}"; shift 2;;
    --bump)    BUMP_MODE="${2:?major|minor|patch}"; shift 2;;
    --stamp)   STAMP=1; shift;;
    -h|--help) usage; exit 0;;
    *) echo "Unknown arg: $1" >&2; usage; exit 2;;
  esac
done

# ----- repo sanity -----
SRC_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$SRC_ROOT" ]] || { echo "Run from inside your source git repo." >&2; exit 1; }
cd "$SRC_ROOT"

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Working tree not clean. Commit or stash first." >&2
  exit 1
fi

# Canonicalize GitHub https URL from 'origin'
REMOTE="$(git remote get-url --push origin)"
GH_HTTPS="$(echo "$REMOTE" \
  | sed -E \
      -e 's#^git@github\.com:(.+)/(.+)\.git$#https://github.com/\1/\2#' \
      -e 's#^ssh://git@github\.com/(.+)/(.+)\.git$#https://github.com/\1/\2#' \
      -e 's#^https://github\.com/(.+)/(.+)(\.git)?$#https://github.com/\1/\2#')"
GH_USER="$(echo "$GH_HTTPS" | sed -E 's#https://github.com/([^/]+)/.*#\1#')"

# ----- version helpers -----
latest_tag_version() {
  # returns X.Y.Z (no 'v'), or 0.0.0 if none
  local t
  t="$(git tag -l 'v[0-9]*' | sed 's/^v//' | sort -V | tail -n1)"
  echo "${t:-0.0.0}"
}

bump_semver() {
  local ver="$1" mode="$2"
  local major minor patch
  IFS=. read -r major minor patch <<<"$ver"
  major=${major:-0}; minor=${minor:-0}; patch=${patch:-0}
  case "$mode" in
    major) echo "$((major+1)).0.0" ;;
    minor) echo "$major.$((minor+1)).0" ;;
    patch|*) echo "$major.$minor.$((patch+1))" ;;
  esac
}

# ----- decide VERSION -----
if [[ $STAMP -eq 1 && -z "$VERSION" ]]; then
  VERSION="$(date +%Y.%m.%d.%H%M)"
fi
if [[ -z "$VERSION" ]]; then
  git fetch --tags --quiet || true
  VERSION="$(bump_semver "$(latest_tag_version)" "$BUMP_MODE")"
fi

TAG="v${VERSION}"
if git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null; then
  echo "Tag ${TAG} already exists. Choose another version." >&2
  exit 1
fi

# ----- tag & push source -----
echo "==> Tagging ${TAG}"
git tag -a "${TAG}" -m "${TAG}"
git push origin "${TAG}"

# ----- sync AUR repo -----
if [[ -d "$AUR_DIR/.git" ]]; then
  echo "==> Updating local AUR repo at $AUR_DIR"
  git -C "$AUR_DIR" fetch origin
  git -C "$AUR_DIR" reset --hard origin/master
else
  echo "==> Cloning AUR repo to $AUR_DIR"
  git clone "$AUR_SSH" "$AUR_DIR"
fi

cd "$AUR_DIR"
[[ -f PKGBUILD ]] || { echo "PKGBUILD missing in $AUR_DIR"; exit 1; }

# ----- bump PKGBUILD -----
echo "==> Bumping AUR PKGBUILD to ${VERSION}-1"
sed -i -E "s|^pkgver=.*$|pkgver=$*
