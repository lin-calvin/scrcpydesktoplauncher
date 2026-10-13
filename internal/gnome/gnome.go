package gnome

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	rootSchema = "org.gnome.desktop.app-folders"
	folderBase = "org.gnome.desktop.app-folders.folder:/org/gnome/desktop/app-folders/folders/"
)

// sessionBus returns a usable session bus address, or "" if none is found.
func sessionBus() string {
	if a := os.Getenv("DBUS_SESSION_BUS_ADDRESS"); a != "" {
		return a
	}
	if r := os.Getenv("XDG_RUNTIME_DIR"); r != "" {
		if _, err := os.Stat(filepath.Join(r, "bus")); err == nil {
			return "unix:path=" + filepath.Join(r, "bus")
		}
	}
	cand := fmt.Sprintf("/run/user/%d/bus", os.Getuid())
	if _, err := os.Stat(cand); err == nil {
		return "unix:path=" + cand
	}
	return ""
}

// envWithBus returns the environment for gsettings, ensuring it talks to the
// real session dconf. Without DBUS_SESSION_BUS_ADDRESS, gsettings silently
// writes to a fallback database that GNOME Shell never reads.
func envWithBus() []string {
	env := os.Environ()
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		if a := sessionBus(); a != "" {
			env = append(env, "DBUS_SESSION_BUS_ADDRESS="+a)
		}
	}
	return env
}

func gsettings(args ...string) (string, error) {
	cmd := exec.Command("gsettings", args...)
	cmd.Env = envWithBus()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gsettings %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Available reports whether the app-folders schema is installed and a session
// bus is reachable (required for GNOME Shell to see the changes).
func Available() bool {
	if os.Getenv("SCRCPYDESKTOP_NO_GNOME") == "1" {
		return false
	}
	if sessionBus() == "" {
		return false
	}
	_, err := gsettings("list-keys", rootSchema)
	return err == nil
}

func folderSchema(id string) string { return folderBase + id + "/" }

// GetChildren returns the current folder-children list.
func GetChildren() ([]string, error) {
	out, err := gsettings("get", rootSchema, "folder-children")
	if err != nil {
		return nil, err
	}
	return parseArray(out), nil
}

func setChildren(ids []string) error {
	_, err := gsettings("set", rootSchema, "folder-children", formatArray(ids))
	return err
}

// EnsureFolder creates/updates the folder `id` with the given display name and
// app list, and makes sure it is registered in folder-children.
func EnsureFolder(id, name string, apps []string, translate bool) error {
	children, err := GetChildren()
	if err != nil {
		return err
	}
	if !contains(children, id) {
		children = append(children, id)
		if err := setChildren(children); err != nil {
			return err
		}
	}
	schema := folderSchema(id)
	if _, err := gsettings("set", schema, "name", quote(name)); err != nil {
		return err
	}
	if _, err := gsettings("set", schema, "translate", fmt.Sprintf("%t", translate)); err != nil {
		return err
	}
	if _, err := gsettings("set", schema, "apps", formatArray(apps)); err != nil {
		return err
	}
	return nil
}

// RemoveFolder unregisters the folder and clears its keys.
func RemoveFolder(id string) error {
	children, err := GetChildren()
	if err != nil {
		return err
	}
	if contains(children, id) {
		var kept []string
		for _, c := range children {
			if c != id {
				kept = append(kept, c)
			}
		}
		if err := setChildren(kept); err != nil {
			return err
		}
	}
	schema := folderSchema(id)
	for _, key := range []string{"name", "translate", "apps", "categories"} {
		_, _ = gsettings("reset", schema, key)
	}
	return nil
}

// --- GVariant string-array helpers (values are safe tokens / simple strings) ---

func parseArray(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@as")
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") {
		return nil
	}
	s = strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
	var out []string
	var cur strings.Builder
	inStr := false
	esc := false
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case esc:
			cur.WriteRune(r)
			esc = false
		case r == '\\' && inStr:
			esc = true
		case r == '\'':
			if inStr {
				flush()
			}
			inStr = !inStr
		case r == ',' && !inStr:
			// separator
		case (r == ' ' || r == '\t') && !inStr:
			// skip
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

func formatArray(items []string) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = quote(it)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func quote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return "'" + s + "'"
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
