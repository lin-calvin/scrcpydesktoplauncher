package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"scrcpydesktoplauncher/internal/config"
	"scrcpydesktoplauncher/internal/daemon"
	"scrcpydesktoplauncher/internal/reconcile"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "gui", "":
		runGUI()
	case "daemon":
		runDaemon()
	case "reconcile", "sync":
		runReconcile()
	case "launch":
		runLaunch(os.Args[2:])
	case "list":
		runList()
	case "clean":
		runClean(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func runGUI() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "Android Launcher",
		Width:            980,
		Height:           720,
		MinWidth:         720,
		MinHeight:        520,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 30, A: 1},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
		os.Exit(1)
	}
}

func runDaemon() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	_ = daemon.Run(ctx)
}

func runReconcile() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	res, err := reconcile.Run(cfg)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("serial=%s total=%d added=%d removed=%d updated=%d\n",
		res.Serial, res.Total, len(res.Added), len(res.Removed), res.Updated)
}

func runLaunch(args []string) {
	if len(args) < 1 {
		fatal(fmt.Errorf("usage: launch <package>"))
	}
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	if err := reconcile.LaunchApp(cfg, args[0]); err != nil {
		os.Exit(1)
	}
}

func runList() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	serial, apps, _, err := reconcile.Enumerate(cfg)
	if err != nil {
		fatal(err)
	}
	selected := reconcile.Filter(cfg, apps)
	fmt.Printf("# %s: %d launchable, %d selected\n", serial, len(apps), len(selected))
	for _, a := range selected {
		kind := "user"
		if a.System {
			kind = "system"
		}
		fmt.Printf("%-8s %-45s %s\n", kind, a.Package, a.Label)
	}
}

func runClean(args []string) {
	keep := false
	for _, a := range args {
		if a == "--keep-folder" {
			keep = true
		}
	}
	if err := reconcile.Clean(keep); err != nil {
		fatal(err)
	}
	fmt.Println("cleaned")
}

func usage() {
	fmt.Print(`scrcpydesktoplauncher - Android apps in the GNOME launcher

Usage:
  scrcpydesktoplauncher            Open the configuration UI
  scrcpydesktoplauncher gui        Open the configuration UI
  scrcpydesktoplauncher daemon     Watch adb and auto sync launcher entries
  scrcpydesktoplauncher reconcile  Sync launcher entries once
  scrcpydesktoplauncher list       List launchable apps (per config)
  scrcpydesktoplauncher launch <pkg>   Open an app in scrcpy (desktop Exec)
  scrcpydesktoplauncher clean [--keep-folder]
`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
