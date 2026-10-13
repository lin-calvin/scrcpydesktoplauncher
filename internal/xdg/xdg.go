package xdg

import (
	"os"
	"path/filepath"

	"scrcpydesktoplauncher/internal/config"
)

func home() string {
	h, _ := os.UserHomeDir()
	return h
}

// DataHome is XDG_DATA_HOME (default ~/.local/share).
func DataHome() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return d
	}
	return filepath.Join(home(), ".local", "share")
}

// StateHome is XDG_STATE_HOME (default ~/.local/state).
func StateHome() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return d
	}
	return filepath.Join(home(), ".local", "state")
}

// CacheHome is XDG_CACHE_HOME (default ~/.cache).
func CacheHome() string {
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return d
	}
	return filepath.Join(home(), ".cache")
}

// RuntimeDir is XDG_RUNTIME_DIR (falls back to a cache subdir).
func RuntimeDir() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return d
	}
	p := filepath.Join(CacheHome(), config.AppName, "run")
	_ = os.MkdirAll(p, 0o700)
	return p
}

// ApplicationsDir is the user .desktop dir watched by GNOME.
func ApplicationsDir() string { return filepath.Join(DataHome(), "applications") }

// IconDir holds extracted per-app PNG icons (Icon= absolute path).
func IconDir() string { return filepath.Join(DataHome(), config.AppName, "icons") }

// StateDir holds injected-state bookkeeping.
func StateDir() string { return filepath.Join(StateHome(), config.AppName) }

// CacheDir holds pulled manifests + device caches.
func CacheDir() string { return filepath.Join(CacheHome(), config.AppName) }

// SocketPath is the daemon control socket.
func SocketPath() string { return filepath.Join(RuntimeDir(), config.AppName+".sock") }

// DesktopPrefix is the filename prefix for generated entries.
const DesktopPrefix = "scrcpydesktoplauncher-"

// FolderID is the dconf folder id under org.gnome.desktop.app-folders.
const FolderID = "android-apps"
