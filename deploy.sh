#!/usr/bin/env bash
set -euo pipefail

# --- config ---
PKGNAME="arch-tui"
OUT_DIR="$(pwd)/out"                            # PKGBUILD + sources live here
PAGES_DIR="$HOME/Developer/arch-setup-tui-repo" # local gh-pages working tree
REMOTE="origin"
BRANCH="gh-pages"
# -------------

usage() {
    cat <<EOF
Usage: $0 [--stamp-ver]
  (no flag)   : bump pkgrel in out/PKGBUILD (…-1 -> …-2) and build
  --stamp-ver : set pkgver=YYYY.MM.DD.HHMM and pkgrel=1, then build
EOF
}
[[ "${1:-}" == "--help" ]] && {
    usage
    exit 0
}

# --- bump version in out/PKGBUILD ---
cd "$OUT_DIR"
STAMP=0
[[ "${1:-}" == "--stamp-ver" ]] && STAMP=1

ver=$(sed -nE 's/^[[:space:]]*pkgver=(.*)$/\1/p' PKGBUILD | tr -d \'\" | head -n1)
rel=$(sed -nE 's/^[[:space:]]*pkgrel=(.*)$/\1/p' PKGBUILD | tr -d \'\" | head -n1)

if [[ $STAMP -eq 1 ]]; then
    new_ver="$(date +%Y.%m.%d.%H%M)"
    new_rel=1
    sed -i -E "s|^[[:space:]]*pkgver=.*$|pkgver=${new_ver}|" PKGBUILD
    sed -i -E "s|^[[:space:]]*pkgrel=.*$|pkgrel=${new_rel}|" PKGBUILD
    echo "==> Set pkgver=${new_ver} pkgrel=${new_rel}"
else
    new_ver="$ver"
    [[ "$rel" =~ ^[0-9]+$ ]] || rel=0
    new_rel=$((rel + 1))
    sed -i -E "s|^[[:space:]]*pkgrel=.*$|pkgrel=${new_rel}|" PKGBUILD
    echo "==> Bump pkgrel: ${ver}-${rel} -> ${ver}-${new_rel}"
fi

# --- build the TUI binary (from repo root) ---
ROOT_DIR="$(
    cd "$OUT_DIR/.."
    pwd
)"
pushd "$ROOT_DIR" >/dev/null
# adjust ./cmd/archsetup if your main package path differs
GOFLAGS=""
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT_DIR/arch-tui" ./cmd/archsetup
popd >/dev/null
chmod +x "$OUT_DIR/arch-tui"

# --- build package ---
echo "==> Building ${PKGNAME} ${new_ver}-${new_rel}"
makepkg -f -C

# capture latest base and debug packages
BASE_PKG="$(ls -1 ${PKGNAME}-[0-9]*-x86_64.pkg.tar.zst | sort -V | tail -n1)"
DBG_PKG="$(ls -1 ${PKGNAME}-debug-[0-9]*-x86_64.pkg.tar.zst 2>/dev/null | sort -V | tail -n1 || true)"
echo "==> Built base:   ${BASE_PKG}"
[[ -n "$DBG_PKG" ]] && echo "==> Built debug: ${DBG_PKG}"

# --- stage into Pages repo ---
mkdir -p "$PAGES_DIR"
cp -f "$BASE_PKG" "$PAGES_DIR/"
[[ -n "$DBG_PKG" ]] && cp -f "$DBG_PKG" "$PAGES_DIR/"

cd "$PAGES_DIR"

# keep last 2 base and last 2 debug packages (avoid 404s if CDN/db caches lag)
ls ${PKGNAME}-[0-9]*-x86_64.pkg.tar.zst 2>/dev/null | sort -V | head -n -2 | xargs -r rm -f
ls ${PKGNAME}-debug-[0-9]*-x86_64.pkg.tar.zst 2>/dev/null | sort -V | head -n -2 | xargs -r rm -f

# rebuild DB with explicit files (base first, then debug if present)
BASE_FILE="$(ls -1 ${PKGNAME}-[0-9]*-x86_64.pkg.tar.zst | sort -V | tail -n1)"
DBG_FILE="$(ls -1 ${PKGNAME}-debug-[0-9]*-x86_64.pkg.tar.zst 2>/dev/null | sort -V | tail -n1 || true)"

if [[ -z "$BASE_FILE" ]]; then
    echo "ERROR: no base package found in $(pwd)" >&2
    exit 1
fi

if [[ -n "$DBG_FILE" ]]; then
    repo-add setup-tui.db.tar.zst "$BASE_FILE" "$DBG_FILE"
else
    repo-add setup-tui.db.tar.zst "$BASE_FILE"
fi

# GitHub Pages: replace symlinks with real files
rm -f setup-tui.db setup-tui.files
cp -f setup-tui.db.tar.zst setup-tui.db
cp -f setup-tui.files.tar.zst setup-tui.files
touch .nojekyll

# --- publish ---
git init -b "$BRANCH" 2>/dev/null || true
git checkout -B "$BRANCH"
git add -A
git commit -m "Publish $(basename "$BASE_FILE")" || true
git push -u "$REMOTE" "$BRANCH"

# --- verify & summary ---
echo "==> Index contains:"
bsdtar -O -xf setup-tui.db.tar.zst --wildcards '*.desc' 2>/dev/null |
    awk '$0=="%FILENAME%"{getline; print "   " $0}'

echo
echo "==> Published files:"
ls -lh setup-tui.db setup-tui.files *.pkg.tar.zst

URL_BASE="$(git remote get-url --push "$REMOTE" | sed -E 's#(git@|https://)github\.com[:/]([^/]+)/([^/]+)\.git#https://\L\2\E.github.io/\3#')"
echo
echo "Pacman URL:   ${URL_BASE}"
