package reconcile

import (
	"sort"

	"scrcpydesktoplauncher/internal/adb"
	"scrcpydesktoplauncher/internal/config"
	"scrcpydesktoplauncher/internal/desktop"
	"scrcpydesktoplauncher/internal/gnome"
	"scrcpydesktoplauncher/internal/helper"
	"scrcpydesktoplauncher/internal/notify"
	"scrcpydesktoplauncher/internal/scrcpy"
	"scrcpydesktoplauncher/internal/xdg"
)

// ErrNoDevice mirrors adb.ErrNoDevice for callers.
var ErrNoDevice = adb.ErrNoDevice

// Result summarizes one reconcile run.
type Result struct {
	Serial   string
	Total    int
	Added    []string
	Removed  []string
	Updated  int
	Selected []helper.App
}

// Filter applies scope/include/exclude to the launchable app list.
func Filter(cfg config.Config, apps []helper.App) []helper.App {
	inc := toSet(cfg.Include)
	exc := toSet(cfg.Exclude)
	var out []helper.App
	for _, a := range apps {
		if exc[a.Package] {
			continue
		}
		if a.System && cfg.Scope == "third_party" && !inc[a.Package] {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// Enumerate resolves the device and dumps the launchable app list.
func Enumerate(cfg config.Config) (string, []helper.App, string, error) {
	serial, err := adb.Resolve(cfg)
	if err != nil {
		return "", nil, "", err
	}
	apps, iconsDir, err := helper.Dump(serial)
	if err != nil {
		return "", nil, "", err
	}
	return serial, apps, iconsDir, nil
}

// Run performs a full add/update/remove pass against the connected device.
func Run(cfg config.Config) (Result, error) {
	var res Result
	serial, apps, iconsDir, err := Enumerate(cfg)
	if err != nil {
		return res, err
	}
	res.Serial = serial

	selected := Filter(cfg, apps)
	res.Selected = selected
	res.Total = len(selected)

	execBin := desktop.ExecPath()

	current, err := desktop.Injected()
	if err != nil {
		return res, err
	}
	have := toSet(current)

	desiredSet := make(map[string]bool, len(selected))
	for _, a := range selected {
		desiredSet[a.Package] = true
		icon := iconsDir + "/" + a.Icon
		if err := desktop.Install(a.Package, a.Label, icon, execBin); err != nil {
			return res, err
		}
		if have[a.Package] {
			res.Updated++
		} else {
			res.Added = append(res.Added, a.Package)
		}
	}

	for _, pkg := range current {
		if !desiredSet[pkg] {
			if err := desktop.Remove(pkg); err != nil {
				return res, err
			}
			res.Removed = append(res.Removed, pkg)
		}
	}

	if cfg.SettingsEntry {
		_ = desktop.InstallSettings("Android Launcher Settings", execBin)
	} else {
		_ = desktop.RemoveSettings()
	}

	if err := syncFolder(cfg, selected); err != nil {
		return res, err
	}
	return res, nil
}

func syncFolder(cfg config.Config, selected []helper.App) error {
	if !gnome.Available() {
		return nil
	}
	ids := make([]string, 0, len(selected))
	for _, a := range selected {
		ids = append(ids, desktop.ID(a.Package))
	}
	sort.Strings(ids)
	if cfg.SettingsEntry {
		ids = append([]string{desktop.SettingsID()}, ids...)
	}
	name := cfg.FolderName
	if name == "" {
		name = "Android"
	}
	return gnome.EnsureFolder(xdg.FolderID, name, ids, false)
}

// LaunchApp resolves the device and opens the given package in scrcpy.
func LaunchApp(cfg config.Config, pkg string) error {
	serial, err := adb.Resolve(cfg)
	if err != nil {
		notify.Error("Android app", err)
		return err
	}
	label := desktop.Label(pkg)
	return scrcpy.Launch(cfg, serial, pkg, label, desktop.WMClass(pkg))
}

// Clean removes every injected entry and the folder.
func Clean(keepFolder bool) error {
	pkgs, err := desktop.Injected()
	if err != nil {
		return err
	}
	for _, p := range pkgs {
		if err := desktop.Remove(p); err != nil {
			return err
		}
	}
	_ = desktop.RemoveSettings()
	if !keepFolder && gnome.Available() {
		return gnome.RemoveFolder(xdg.FolderID)
	}
	return nil
}

func toSet(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, s := range list {
		m[s] = true
	}
	return m
}
