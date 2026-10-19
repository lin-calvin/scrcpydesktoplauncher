#!/usr/bin/env bash
# Build a patched scrcpy client (physical-pixel flex display) and assemble a
# ready-to-bundle directory at build/scrcpy-patched/.
#
# Usage:
#   ./packaging/build-scrcpy.sh            # fast: system SDL3/ffmpeg (meson)
#   STATIC=1 ./packaging/build-scrcpy.sh   # static: release/build_linux.sh (CI)
#
# Env:
#   SCRCPY_VERSION  upstream tag to build (default v5.0.1)
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/.." && pwd)"
VERSION="${SCRCPY_VERSION:-v5.0.1}"
OUT="$root/build/scrcpy-patched"
WORK="$root/build/scrcpy-build"
SRC="$WORK/src"

mkdir -p "$WORK"

# --- official dist (server jar, adb, icons, man, LICENSE) ---
official="$root/build/scrcpy-linux-x86_64-$VERSION"
if [ ! -x "$official/scrcpy" ]; then
	tarball="$root/build/scrcpy-$VERSION.tar.gz"
	if [ ! -f "$tarball" ]; then
		curl -fL "https://github.com/Genymobile/scrcpy/releases/download/$VERSION/scrcpy-linux-x86_64-$VERSION.tar.gz" -o "$tarball"
	fi
	tar xzf "$tarball" -C "$root/build"
fi

# --- source + patch ---
if [ ! -d "$SRC/.git" ]; then
	git clone -q --depth 1 --branch "$VERSION" https://github.com/Genymobile/scrcpy "$SRC"
fi
if ! git -C "$SRC" diff --quiet; then
	echo "working tree dirty, resetting" >&2
	git -C "$SRC" checkout -- .
fi
git -C "$SRC" apply "$here/patches/scrcpy-flex-physical.patch"

# --- build client ---
if [ "${STATIC:-0}" = "1" ]; then
	( cd "$SRC" && release/build_linux.sh x86_64 )
	patched="$SRC/release/work/build-linux-x86_64/dist/scrcpy"
else
	rm -rf "$SRC/build"
	( cd "$SRC" && meson setup build --buildtype=release -Dcompile_server=false -Dportable=true >/dev/null
	  ninja -C build >/dev/null )
	patched="$SRC/build/app/scrcpy"
fi
[ -x "$patched" ] || { echo "build failed: $patched missing" >&2; exit 1; }

# --- assemble: official dist with the patched client ---
rm -rf "$OUT"
cp -r "$official" "$OUT"
install -m 0755 "$patched" "$OUT/scrcpy"

echo "patched scrcpy -> $OUT"
"$OUT/scrcpy" --version | head -1
