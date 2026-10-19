package scrcpy

import (
	"testing"

	"scrcpydesktoplauncher/internal/config"
)

func has(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func TestBuildArgsDefaults(t *testing.T) {
	cfg := config.Default() // audio on, flex on, keep content on
	args := BuildArgs(cfg, "SERIAL", "com.foo.bar", "Foo")

	if !has(args, "-s") || !has(args, "SERIAL") {
		t.Errorf("missing -s SERIAL: %v", args)
	}
	if !has(args, "--new-display") {
		t.Errorf("missing --new-display: %v", args)
	}
	if !has(args, "--flex-display") {
		t.Errorf("missing --flex-display: %v", args)
	}
	if has(args, "--no-vd-destroy-content") {
		t.Errorf("app must not be moved to the phone screen by default: %v", args)
	}
	if !has(args, "--start-app=+com.foo.bar") {
		t.Errorf("missing --start-app: %v", args)
	}
	if has(args, "--no-audio") {
		t.Errorf("audio should be forwarded by default: %v", args)
	}
}

func TestBuildArgsAudioOffAndSize(t *testing.T) {
	cfg := config.Default()
	cfg.Audio = false
	cfg.Scrcpy.KeepContentOnClose = true
	cfg.Scrcpy.Size = "1080x1920"
	cfg.Scrcpy.DPI = "420"
	cfg.Scrcpy.BitRate = "16M"
	cfg.Scrcpy.FlexDisplay = false

	args := BuildArgs(cfg, "", "pkg", "Pkg Name")
	if !has(args, "--no-audio") {
		t.Errorf("expected --no-audio: %v", args)
	}
	if !has(args, "--video-bit-rate=16M") {
		t.Errorf("expected video bitrate: %v", args)
	}
	if !has(args, "--no-vd-destroy-content") {
		t.Errorf("expected keep-content flag when enabled: %v", args)
	}
	if !has(args, "--window-title=Pkg Name") {
		t.Errorf("expected window title: %v", args)
	}
	if !has(args, "--new-display=1080x1920/420") {
		t.Errorf("expected sized new display: %v", args)
	}
	if has(args, "--flex-display") {
		t.Errorf("flex should be off: %v", args)
	}
	if has(args, "-s") {
		t.Errorf("no serial expected: %v", args)
	}
}

func TestBuildArgsHiDPIScale(t *testing.T) {
	cfg := config.Default()
	cfg.Scrcpy.HiDPIScale = 2
	cfg.Scrcpy.Size = "640x480"
	cfg.Scrcpy.FlexDisplay = true // must be suppressed in physical mode

	args := BuildArgs(cfg, "", "pkg", "")
	if !has(args, "--new-display=1280x960") {
		t.Errorf("expected physical display size: %v", args)
	}
	if !has(args, "--window-width=640") || !has(args, "--window-height=480") {
		t.Errorf("expected logical window size: %v", args)
	}
	if !has(args, "--flex-display") {
		t.Errorf("flex must stay enabled (patched physical mode): %v", args)
	}
}
