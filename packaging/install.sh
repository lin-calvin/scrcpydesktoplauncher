#!/usr/bin/env bash
# Install the built binary + systemd --user unit, then do an initial sync.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/.." && pwd)"
bin_dir="$HOME/.local/bin"
bin="$bin_dir/scrcpydesktoplauncher"
built="$root/build/bin/scrcpydesktoplauncher"

command -v wails >/dev/null 2>&1 || export PATH="$HOME/go/bin:$PATH"

if [ ! -x "$built" ]; then
	echo "building…"
	( cd "$root" && wails build )
fi

mkdir -p "$bin_dir"
install -m 0755 "$built" "$bin"
echo "installed binary: $bin"

unit_dir="$HOME/.config/systemd/user"
mkdir -p "$unit_dir"
install -m 0644 "$here/scrcpydesktoplauncher.service" \
	"$unit_dir/scrcpydesktoplauncher.service"

if command -v systemctl >/dev/null 2>&1; then
	systemctl --user daemon-reload || true
	systemctl --user enable --now scrcpydesktoplauncher.service || true
	echo "systemd unit: scrcpydesktoplauncher.service (enabled)"
fi

echo "initial sync…"
"$bin" reconcile || true

cat <<EOF

Done.
  GUI     : $bin gui
  sync    : $bin reconcile
  remove  : $bin clean
  logs    : journalctl --user -u scrcpydesktoplauncher -f
EOF
