package scrcpy

import (
	"os"
	"os/exec"
	"path/filepath"

	"scrcpydesktoplauncher/internal/config"
)

// binary resolves the scrcpy executable: a sibling of our own binary if
// present (e.g. bundled inside an AppImage), otherwise "scrcpy" from PATH.
func binary() string {
	if exe, err := os.Executable(); err == nil {
		cand := filepath.Join(filepath.Dir(exe), "scrcpy")
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand
		}
	}
	return "scrcpy"
}

// BuildArgs assembles the scrcpy command line for launching one Android app on
// a new virtual display. title sets the window title (empty = scrcpy default).
func BuildArgs(cfg config.Config, serial, pkg, title string) []string {
	nd := "--new-display"
	switch {
	case cfg.Scrcpy.Size != "" && cfg.Scrcpy.DPI != "":
		nd = "--new-display=" + cfg.Scrcpy.Size + "/" + cfg.Scrcpy.DPI
	case cfg.Scrcpy.Size != "":
		nd = "--new-display=" + cfg.Scrcpy.Size
	case cfg.Scrcpy.DPI != "":
		nd = "--new-display=/" + cfg.Scrcpy.DPI
	}
	args := []string{nd}

	if cfg.Scrcpy.FlexDisplay {
		args = append(args, "--flex-display")
	}
	if cfg.Scrcpy.NoVDDecorations {
		args = append(args, "--no-vd-system-decorations")
	}
	if cfg.Scrcpy.KeepContentOnClose {
		args = append(args, "--no-vd-destroy-content")
	}
	if !cfg.Audio {
		args = append(args, "--no-audio")
	}
	if title != "" {
		args = append(args, "--window-title="+title)
	}
	args = append(args, "--start-app=+"+pkg)

	if serial != "" {
		args = append([]string{"-s", serial}, args...)
	}
	return args
}

// Launch runs scrcpy for the given package and blocks until it exits.
// appID (when set) becomes the window app-id/WM_CLASS so the compositor matches
// the window back to the app's desktop entry and shows its icon.
func Launch(cfg config.Config, serial, pkg, title, appID string) error {
	args := BuildArgs(cfg, serial, pkg, title)
	cmd := exec.Command(binary(), args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	// SDL_APP_ID drives the Wayland app-id (and X11 WM_CLASS) used by the
	// compositor to match the window back to the app's .desktop (icon/title).
	// Do NOT set SDL_APP_NAME: it is a reserved hint scrcpy also sets, which
	// makes SDL log "Could not set app name" and mis-associates the window.
	if appID != "" {
		cmd.Env = append(cmd.Env, "SDL_APP_ID="+appID)
	}
	return cmd.Run()
}
