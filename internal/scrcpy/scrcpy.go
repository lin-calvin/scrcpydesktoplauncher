package scrcpy

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
//
// HiDPI: scrcpy's window is DPI-aware, so by default --flex-display sizes the
// virtual display in *logical* pixels and the compositor upscales the video
// (blurry at 200%). When HiDPIScale > 1 we create the display at physical
// pixels, pin the window to the logical base size, and set
// SCRCPY_FLEX_PHYSICAL=1 so a patched scrcpy resizes the display from the
// window's physical pixel size (see packaging/patches/).
// hasPatchedScrcpy reports whether a scrcpy binary sits next to ours (i.e. the
// AppImage bundles the patched client). Overridable in tests.
var hasPatchedScrcpy = func() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	fi, err := os.Stat(filepath.Join(filepath.Dir(exe), "scrcpy"))
	return err == nil && !fi.IsDir()
}

func BuildArgs(cfg config.Config, serial, pkg, title string) []string {
	scale := cfg.Scrcpy.HiDPIScale
	if scale <= 0 {
		scale = 1
	}
	physical := scale > 1 && hasPatchedScrcpy()

	baseW, baseH := parseSize(cfg.Scrcpy.Size)
	if baseW == 0 || baseH == 0 {
		baseW, baseH = 1280, 960 // scrcpy default new-display size
	}

	flex := cfg.Scrcpy.FlexDisplay
	var nd string
	if scale > 1 {
		if !physical {
			// Unpatched scrcpy: flex would override the physical size, so pin
			// a fixed display and let the window scale it down to logical size.
			flex = false
		}
		nd = fmt.Sprintf("--new-display=%dx%d",
			int(math.Round(float64(baseW)*scale)),
			int(math.Round(float64(baseH)*scale)))
		if cfg.Scrcpy.DPI != "" {
			nd += "/" + cfg.Scrcpy.DPI
		}
	} else {
		nd = "--new-display"
		switch {
		case cfg.Scrcpy.Size != "" && cfg.Scrcpy.DPI != "":
			nd = "--new-display=" + cfg.Scrcpy.Size + "/" + cfg.Scrcpy.DPI
		case cfg.Scrcpy.Size != "":
			nd = "--new-display=" + cfg.Scrcpy.Size
		case cfg.Scrcpy.DPI != "":
			nd = "--new-display=/" + cfg.Scrcpy.DPI
		}
	}
	args := []string{nd}

	if scale > 1 && !physical {
		// Only valid without flex: keep the window at the logical size so the
		// physical video is downscaled to it.
		args = append(args,
			fmt.Sprintf("--window-width=%d", baseW),
			fmt.Sprintf("--window-height=%d", baseH))
	}
	if flex {
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
	if cfg.Scrcpy.BitRate != "" {
		args = append(args, "--video-bit-rate="+cfg.Scrcpy.BitRate)
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

// parseSize parses "WxH" into integers, returning zeros when absent/invalid.
func parseSize(s string) (int, int) {
	if s == "" {
		return 0, 0
	}
	parts := strings.SplitN(strings.ToLower(s), "x", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0
	}
	return w, h
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
	// Ask the patched scrcpy to resize flex displays using physical pixels.
	if cfg.Scrcpy.HiDPIScale > 1 && hasPatchedScrcpy() {
		cmd.Env = append(cmd.Env, "SCRCPY_FLEX_PHYSICAL=1")
	}
	return cmd.Run()
}
