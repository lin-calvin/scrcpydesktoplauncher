#!/usr/bin/env bash
# Rebuild the on-device helper dex and copy it into internal/helper/.
# Requires: JDK (javac) + Android SDK build-tools (d8) + platforms android.jar.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/.." && pwd)"

SDK="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-$HOME/Android/Sdk}}"
API="${API:-35}"

JAR="$SDK/platforms/android-$API/android.jar"
D8="$(ls "$SDK"/build-tools/*/d8 2>/dev/null | sort -V | tail -1)"

[ -f "$JAR" ] || { echo "android.jar not found: $JAR" >&2; exit 1; }
[ -n "$D8" ]  || { echo "d8 not found under $SDK/build-tools" >&2; exit 1; }

out="$here/out"
rm -rf "$out"; mkdir -p "$out/classes"

echo "javac <- $JAR"
javac --release 11 -classpath "$JAR" -d "$out/classes" \
    "$here/src/com/calvin/alauncher/AppList.java"

echo "d8 -> classes.dex   ($D8)"
"$D8" --release --lib "$JAR" --output "$out" "$out/classes/com/calvin/alauncher/AppList.class"

mkdir -p "$root/internal/helper"
cp "$out/classes.dex" "$root/internal/helper/helper.dex"
echo "wrote $root/internal/helper/helper.dex"
