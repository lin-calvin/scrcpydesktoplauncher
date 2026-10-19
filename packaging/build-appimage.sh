#!/usr/bin/env bash
# Build an AppImage for scrcpydesktoplauncher, bundling scrcpy (client +
# scrcpy-server + adb) so the user only needs the host WebKit/GTK runtime.
#
# Produces: build/scrcpydesktoplauncher-x86_64.AppImage
#
# Env:
#   SCRCPY_VERSION   scrcpy release to bundle (default: pinned below)
#   SCRCPY_DIR       use an already-extracted scrcpy dir instead of downloading
#   BUNDLE_GTK=1     also bundle GTK/WebKit via linuxdeploy-plugin-gtk (if installed)
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/.." && pwd)"
APP=scrcpydesktoplauncher
SCRCPY_VERSION="${SCRCPY_VERSION:-v5.0.1}"

command -v appimagetool >/dev/null 2>&1 || { echo "appimagetool not found" >&2; exit 1; }
export APPIMAGE_EXTRACT_AND_RUN=1

[ -x "$root/build/bin/$APP" ] || { echo "building wails app…"; ( cd "$root" && wails build ); }

# --- locate scrcpy (prefer a patched build, fall back to the official tarball) ---
scrcpy_dir="${SCRCPY_DIR:-$root/build/scrcpy-patched}"
if [ ! -x "$scrcpy_dir/scrcpy" ]; then
	scrcpy_dir="$root/build/scrcpy-linux-x86_64-$SCRCPY_VERSION"
	if [ ! -x "$scrcpy_dir/scrcpy" ]; then
		url="https://github.com/Genymobile/scrcpy/releases/download/$SCRCPY_VERSION/scrcpy-linux-x86_64-$SCRCPY_VERSION.tar.gz"
		tmp="$root/build/scrcpy-$SCRCPY_VERSION.tar.gz"
		echo "downloading $url"
		curl -fL "$url" -o "$tmp"
		tar xzf "$tmp" -C "$root/build"
	fi
fi
[ -x "$scrcpy_dir/scrcpy" ] || { echo "scrcpy not found in $scrcpy_dir" >&2; exit 1; }
echo "bundling scrcpy from $scrcpy_dir"

# --- assemble AppDir ---
appdir="$root/build/appimage/$APP.AppDir"
rm -rf "$appdir"
mkdir -p "$appdir/usr/bin" \
         "$appdir/usr/share/applications" \
         "$appdir/usr/share/icons/hicolor/256x256/apps"

install -m 0755 "$root/build/bin/$APP" "$appdir/usr/bin/$APP"
install -m 0755 "$here/AppRun" "$appdir/AppRun"

# scrcpy payload (scrcpy finds scrcpy-server next to itself)
for f in scrcpy scrcpy-server adb scrcpy.png disconnected.png; do
	[ -f "$scrcpy_dir/$f" ] && install -m 0755 "$scrcpy_dir/$f" "$appdir/usr/bin/$f"
done

# icon (256x256)
if command -v convert >/dev/null 2>&1; then
	convert "$root/build/appicon.png" -resize 256x256 "$appdir/$APP.png"
else
	cp "$root/build/appicon.png" "$appdir/$APP.png"
fi
cp "$appdir/$APP.png" "$appdir/usr/share/icons/hicolor/256x256/apps/$APP.png"
cp "$appdir/$APP.png" "$appdir/.DirIcon"

# desktop entry
cat > "$appdir/$APP.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Android Launcher
Comment=Run Android apps from the GNOME launcher via scrcpy
Exec=$APP
Icon=$APP
Terminal=false
Categories=Utility;Settings;
StartupWMClass=scrcpydesktoplauncher
EOF
cp "$appdir/$APP.desktop" "$appdir/usr/share/applications/$APP.desktop"

# --- optional: bundle GTK/WebKit for full portability ---
if [ "${BUNDLE_GTK:-0}" = "1" ] && command -v linuxdeploy >/dev/null 2>&1; then
	DEPLOY_GTK_VERSION=3 linuxdeploy \
		--appdir "$appdir" \
		--desktop-file "$appdir/$APP.desktop" \
		--icon-file "$appdir/$APP.png" \
		--plugin gtk
fi

# --- package ---
out="$root/build/$APP-x86_64.AppImage"
rm -f "$out"
ARCH=x86_64 appimagetool "$appdir" "$out"
echo "built: $out ($(du -h "$out" | cut -f1))"
