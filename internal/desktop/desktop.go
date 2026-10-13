package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"scrcpydesktoplauncher/internal/xdg"
)

// ExecPath is the absolute path used in Exec=. Inside an AppImage this is the
// AppImage file itself (the mounted /tmp path would be gone after exit).
func ExecPath() string {
	if ai := os.Getenv("APPIMAGE"); ai != "" {
		return ai
	}
	p, err := os.Executable()
	if err != nil {
		return "scrcpydesktoplauncher"
	}
	if rp, err := filepath.EvalSymlinks(p); err == nil {
		return rp
	}
	return p
}

// ID returns the .desktop basename for a package.
func ID(pkg string) string { return xdg.DesktopPrefix + pkg + ".desktop" }

// WMClass is the window app-id (Wayland) / WM_CLASS (X11) used to match a
// scrcpy window back to this desktop entry, so GNOME shows the app icon.
func WMClass(pkg string) string { return xdg.DesktopPrefix + pkg }

// Path returns the absolute .desktop path for a package.
func Path(pkg string) string { return filepath.Join(xdg.ApplicationsDir(), ID(pkg)) }

// Label reads the Name= of the installed entry, falling back to the package.
func Label(pkg string) string {
	b, err := os.ReadFile(Path(pkg))
	if err != nil {
		return pkg
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "Name=") {
			return strings.TrimPrefix(line, "Name=")
		}
	}
	return pkg
}

// SettingsID is the desktop id for the config UI entry.
func SettingsID() string { return xdg.DesktopPrefix + "settings.desktop" }

// Install writes the icon and .desktop entry for an app.
func Install(pkg, label, iconSrc, execBin string) error {
	if err := os.MkdirAll(xdg.IconDir(), 0o755); err != nil {
		return err
	}
	iconDst := filepath.Join(xdg.IconDir(), pkg+".png")
	if err := copyFile(iconSrc, iconDst); err != nil {
		return err
	}
	if err := os.MkdirAll(xdg.ApplicationsDir(), 0o755); err != nil {
		return err
	}
	content := buildDesktop(label, execBin, iconDst, pkg, WMClass(pkg))
	return writeAtomic(Path(pkg), content)
}

// InstallSettings writes the "Android Settings" entry that opens the GUI.
func InstallSettings(label, execBin string) error {
	if err := os.MkdirAll(xdg.ApplicationsDir(), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=%s
Comment=Configure Android apps in the launcher
Exec=%s gui
Icon=preferences-system
Terminal=false
Categories=Android;Settings;
StartupWMClass=scrcpydesktoplauncher
`, label, quoteExec(execBin))
	return writeAtomic(filepath.Join(xdg.ApplicationsDir(), SettingsID()), content)
}

// Remove deletes the .desktop entry (and icon) for a package.
func Remove(pkg string) error {
	_ = os.Remove(filepath.Join(xdg.IconDir(), pkg+".png"))
	err := os.Remove(Path(pkg))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// RemoveSettings deletes the settings entry.
func RemoveSettings() error {
	err := os.Remove(filepath.Join(xdg.ApplicationsDir(), SettingsID()))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Injected returns the package names of all generated entries on disk.
func Injected() ([]string, error) {
	entries, err := os.ReadDir(xdg.ApplicationsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var pkgs []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, xdg.DesktopPrefix) || !strings.HasSuffix(name, ".desktop") {
			continue
		}
		pkg := strings.TrimSuffix(strings.TrimPrefix(name, xdg.DesktopPrefix), ".desktop")
		if pkg == "settings" {
			continue
		}
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)
	return pkgs, nil
}

func buildDesktop(label, execBin, iconPath, pkg, wmclass string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=%s
Comment=Android app (%s)
Exec=%s launch %s
Icon=%s
Terminal=false
Categories=Android;
StartupWMClass=%s
`, label, pkg, quoteExec(execBin), pkg, iconPath, wmclass)
}

func quoteExec(s string) string {
	if strings.ContainsAny(s, " \t\"'\\") {
		s = strings.ReplaceAll(s, `"`, `\"`)
		return `"` + s + `"`
	}
	return s
}

func writeAtomic(path, content string) error {
	if cur, err := os.ReadFile(path); err == nil && string(cur) == content {
		return nil // unchanged: avoid churn that can race GNOME app matching
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if cur, err := os.ReadFile(dst); err == nil && bytesEqual(cur, b) {
		return nil
	}
	return os.WriteFile(dst, b, 0o644)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
