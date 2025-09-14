#!/usr/bin/env bash
set -euo pipefail

AUR_PKG="${AUR_PKG:-bas-tui}"
AUR_DIR="${AUR_DIR:-$HOME/tmp/$AUR_PKG}"
AUR_SSH="${AUR_SSH:-ssh://aur@aur.archlinux.org/${AUR_PKG}.git}"
BUMP_MODE="${BUMP_MODE:-patch}"
BIN_NAME="${BIN_NAME:-bas-tui}"

usage() {
  cat <<EOF
Usage:
  $0 [--version X.Y.Z] [--bump major|minor|patch] [--stamp] [--dry-run]
Env: AUR_PKG, AUR_DIR, AUR_SSH, BUMP_MODE, BIN_NAME
EOF
}

VERSION=""
STAMP=0
DRYRUN=0
while [[ $# -gt 0 ]]; do
  case "$1" in
  --version)
    VERSION="${2:?}"
    shift 2
    ;;
  --bump)
    BUMP_MODE="${2:?}"
    shift 2
    ;;
  --stamp)
    STAMP=1
    shift
    ;;
  --dry-run)
    DRYRUN=1
    shift
    ;;
  -h | --help)
    usage
    exit 0
    ;;
  *)
    echo "unknown arg: $1" >&2
    usage
    exit 2
    ;;
  esac
done

# --- repo roots & remotes ---
ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$ROOT" ]] || {
  echo "run inside git repo"
  exit 1
}
cd "$ROOT"

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Working tree not clean. Commit or stash first." >&2
  exit 1
fi

REMOTE="$(git remote get-url --push origin)"
GH_HTTPS="$(echo "$REMOTE" |
  sed -E \
    -e 's#^git@github\.com:(.+)/(.+)\.git$#https://github.com/\1/\2#' \
    -e 's#^ssh://git@github\.com/(.+)/(.+)\.git$#https://github.com/\1/\2#' \
    -e 's#^https://github\.com/(.+)/(.+)(\.git)?$#https://github.com/\1/\2#')"
REPO_SLUG="${GH_HTTPS#https://github.com/}"

latest_tag_version() { git tag -l 'v[0-9]*' | sed 's/^v//' | sort -V | tail -n1; }
bump_semver() {
  IFS=. read -r M m p <<<"${1:-0.0.0}"
  case "$2" in
  major) echo "$((M + 1)).0.0" ;;
  minor) echo "$M.$((m + 1)).0" ;;
  *) echo "$M.$m.$((p + 1))" ;;
  esac
}

# --- decide VERSION/TAG idempotently ---
git fetch --tags --quiet || true
HEAD_TAG="$(git tag --points-at HEAD | grep -E '^v[0-9]+' || true)"

if [[ $STAMP -eq 1 && -z "$VERSION" ]]; then
  VERSION="$(date +%Y.%m.%d.%H%M)"
fi
if [[ -z "$VERSION" ]]; then
  if [[ -n "$HEAD_TAG" ]]; then
    VERSION="${HEAD_TAG#v}"
  else
    BASE="$(latest_tag_version)"
    BASE="${BASE:-0.0.0}"
    VERSION="$(bump_semver "$BASE" "$BUMP_MODE")"
  fi
fi
TAG="v${VERSION}"

# --- tag only if it doesn't exist ---
if ! git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null; then
  echo "==> Tagging ${TAG}"
  ((DRYRUN)) || {
    git tag -a "${TAG}" -m "${TAG}"
    git push origin "${TAG}"
  }
else
  echo "==> Reusing existing tag ${TAG}"
fi

# ---------------- Build binaries ----------------
ART_DIR="${ROOT}/dist/${TAG}"
mkdir -p "$ART_DIR"

build_one() {
  local GOOS="$1" GOARCH="$2"
  local OUT="${ART_DIR}/${BIN_NAME}-${TAG}-${GOOS}-${GOARCH}"
  echo "==> building $OUT"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags="-s -w" -o "$OUT" ./cmd/archsetup
  chmod +x "$OUT"
}

build_one linux amd64
build_one linux arm64

package_one() {
  local f="$1"
  (cd "$(dirname "$f")" && tar -czf "${f}.tar.gz" "$(basename "$f")")
}
package_one "${ART_DIR}/${BIN_NAME}-${TAG}-linux-amd64"
package_one "${ART_DIR}/${BIN_NAME}-${TAG}-linux-arm64"

# ---------------- GitHub Release ----------------
need_gh() { command -v gh >/dev/null 2>&1 || {
  echo "Install GitHub CLI (gh)"
  exit 1
}; }
need_gh

release_exists() { gh release view "$TAG" --repo "$REPO_SLUG" >/dev/null 2>&1; }

if release_exists; then
  echo "==> Release ${TAG} exists — uploading/overwriting assets"
else
  echo "==> Creating GitHub release ${TAG}"
  ((DRYRUN)) || gh release create "$TAG" --repo "$REPO_SLUG" --title "$TAG" --notes "$TAG"
fi

echo "==> Uploading assets"
# Upload versioned assets (binaries and/or tarballs)
((DRYRUN)) || gh release upload "$TAG" \
  "${ART_DIR}/${BIN_NAME}-${TAG}-linux-amd64" \
  "${ART_DIR}/${BIN_NAME}-${TAG}-linux-arm64" \
  "${ART_DIR}/${BIN_NAME}-${TAG}-linux-amd64.tar.gz" \
  "${ART_DIR}/${BIN_NAME}-${TAG}-linux-arm64.tar.gz" \
  --repo "$REPO_SLUG" --clobber

# Also upload stable (version-less) copies for your installer to fetch
cp "${ART_DIR}/${BIN_NAME}-${TAG}-linux-amd64" "${ART_DIR}/${BIN_NAME}-linux-amd64"
cp "${ART_DIR}/${BIN_NAME}-${TAG}-linux-arm64" "${ART_DIR}/${BIN_NAME}-linux-arm64"
# cp "${ART_DIR}/${BIN_NAME}-${TAG}-linux-amd64.tar.gz" "${ART_DIR}/${BIN_NAME}-linux-amd64.tar.gz"
# cp "${ART_DIR}/${BIN_NAME}-${TAG}-linux-arm64.tar.gz" "${ART_DIR}/${BIN_NAME}-linux-arm64.tar.gz"

((DRYRUN)) || gh release upload "$TAG" \
  "${ART_DIR}/${BIN_NAME}-linux-amd64" \
  "${ART_DIR}/${BIN_NAME}-linux-arm64" \
  --repo "$REPO_SLUG" --clobber

# ---------------- AUR bump/push ----------------
sync_aur() {
  if [[ -d "$AUR_DIR/.git" ]]; then
    echo "==> Updating local AUR repo at $AUR_DIR"
    git -C "$AUR_DIR" fetch origin
    git -C "$AUR_DIR" reset --hard origin/master
  else
    echo "==> Cloning AUR repo to $AUR_DIR"
    git clone "$AUR_SSH" "$AUR_DIR"
  fi

  cd "$AUR_DIR"
  [[ -f PKGBUILD ]] || {
    echo "PKGBUILD missing in $AUR_DIR"
    exit 1
  }

  echo "==> Bumping AUR PKGBUILD to ${VERSION}-1"
  sed -i -E "s|^pkgver=.*$|pkgver=${VERSION}|" PKGBUILD
  sed -i -E "s|^pkgrel=.*$|pkgrel=1|" PKGBUILD
  sed -i -E "s|^url=.*$|url=\"${GH_HTTPS}\"|" PKGBUILD

  if command -v updpkgsums >/dev/null 2>&1; then updpkgsums; fi
  makepkg --printsrcinfo >.SRCINFO

  git config user.name "${GIT_AUTHOR_NAME:-DarkBones}"
  git config user.email "${GIT_AUTHOR_EMAIL:-$(whoami)@users.noreply.github.com}"
  git add PKGBUILD .SRCINFO
  if ! git diff --cached --quiet; then
    git commit -m "${AUR_PKG} ${VERSION}-1"
    ((DRYRUN)) || git push origin HEAD:master
  else
    echo "AUR repo already up to date for ${VERSION}"
  fi
}
sync_aur

echo "==> Done. Assets in: ${ART_DIR}"
