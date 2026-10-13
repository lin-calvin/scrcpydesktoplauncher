package main

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"

	"scrcpydesktoplauncher/internal/adb"
	"scrcpydesktoplauncher/internal/config"
	"scrcpydesktoplauncher/internal/reconcile"
	"scrcpydesktoplauncher/internal/xdg"
)

// App is the Wails-bound backend.
type App struct {
	ctx context.Context
}

// NewApp creates the app.
func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// Status is device connection state shown in the UI.
type Status struct {
	Connected bool     `json:"connected"`
	Devices   []string `json:"devices"`
}

// AppView is one row in the app selection list.
type AppView struct {
	Package  string `json:"package"`
	Label    string `json:"label"`
	System   bool   `json:"system"`
	Selected bool   `json:"selected"`
}

// Result mirrors reconcile.Result for JSON transport.
type Result struct {
	Serial  string   `json:"serial"`
	Total   int      `json:"total"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Updated int      `json:"updated"`
}

// GetConfig returns the current config.
func (a *App) GetConfig() (config.Config, error) { return config.Load() }

// SaveConfig persists the config.
func (a *App) SaveConfig(cfg config.Config) error { return config.Save(cfg) }

// DeviceStatus reports connected adb devices.
func (a *App) DeviceStatus() Status {
	devs, _ := adb.List()
	var serials []string
	for _, d := range devs {
		if d.State == "device" {
			serials = append(serials, d.Serial)
		}
	}
	return Status{Connected: len(serials) > 0, Devices: serials}
}

// ListApps enumerates launchable apps and marks the ones selected by config.
func (a *App) ListApps() ([]AppView, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	_, apps, _, err := reconcile.Enumerate(cfg)
	if err != nil {
		return nil, err
	}
	sel := reconcile.Filter(cfg, apps)
	selSet := make(map[string]bool, len(sel))
	for _, s := range sel {
		selSet[s.Package] = true
	}
	out := make([]AppView, 0, len(apps))
	for _, ap := range apps {
		out = append(out, AppView{
			Package:  ap.Package,
			Label:    ap.Label,
			System:   ap.System,
			Selected: selSet[ap.Package],
		})
	}
	return out, nil
}

// IconDataURL returns a base64 data URL for an app icon.
func (a *App) IconDataURL(pkg string) string {
	candidates := []string{filepath.Join(xdg.IconDir(), pkg+".png")}
	if matches, _ := filepath.Glob(filepath.Join(xdg.CacheDir(), "*", "dump", "icons", pkg+".png")); len(matches) > 0 {
		candidates = append(candidates, matches...)
	}
	for _, p := range candidates {
		if b, err := os.ReadFile(p); err == nil {
			return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)
		}
	}
	return ""
}

// ReconcileNow runs a sync pass and returns the summary.
func (a *App) ReconcileNow() (Result, error) {
	cfg, err := config.Load()
	if err != nil {
		return Result{}, err
	}
	res, err := reconcile.Run(cfg)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Serial:  res.Serial,
		Total:   res.Total,
		Added:   res.Added,
		Removed: res.Removed,
		Updated: res.Updated,
	}, nil
}

// CleanAll removes all injected entries and the folder.
func (a *App) CleanAll() error { return reconcile.Clean(false) }
