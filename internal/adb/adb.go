package adb

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"scrcpydesktoplauncher/internal/config"
)

// exe resolves the adb executable: a sibling of our own binary if present
// (e.g. bundled inside an AppImage), otherwise "adb" from PATH.
func exe() string {
	if e, err := os.Executable(); err == nil {
		cand := filepath.Join(filepath.Dir(e), "adb")
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand
		}
	}
	return "adb"
}

// ErrNoDevice is returned when no usable adb device is attached.
var ErrNoDevice = errors.New("no adb device connected")

// Device is one entry from `adb devices`.
type Device struct {
	Serial string
	State  string
}

// StartServer ensures the adb server is running.
func StartServer() error {
	return exec.Command(exe(), "start-server").Run()
}

// List returns the currently known devices.
func List() ([]Device, error) {
	out, err := exec.Command(exe(), "devices").Output()
	if err != nil {
		return nil, err
	}
	var devs []Device
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		devs = append(devs, Device{Serial: f[0], State: f[1]})
	}
	return devs, nil
}

func online(devs []Device) []Device {
	var r []Device
	for _, d := range devs {
		if d.State == "device" {
			r = append(r, d)
		}
	}
	return r
}

// Resolve picks the device serial to use, honoring cfg.Device and transport pref.
func Resolve(cfg config.Config) (string, error) {
	if cfg.Device != "" && cfg.Device != "auto" {
		return cfg.Device, nil
	}
	devs, err := List()
	if err != nil {
		return "", err
	}
	devs = online(devs)
	if len(devs) == 0 {
		return "", ErrNoDevice
	}
	isUSB := func(s string) bool { return !strings.Contains(s, ":") }

	switch cfg.PreferTransport {
	case "usb":
		for _, d := range devs {
			if isUSB(d.Serial) {
				return d.Serial, nil
			}
		}
	case "tcp":
		for _, d := range devs {
			if !isUSB(d.Serial) {
				return d.Serial, nil
			}
		}
	}
	return devs[0].Serial, nil
}

// Shell runs a raw shell command line on the device.
func Shell(serial, cmdline string) (string, error) {
	out, err := exec.Command(exe(), "-s", serial, "shell", cmdline).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("adb shell: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// Push copies a local file to the device.
func Push(serial, local, remote string) error {
	out, err := exec.Command(exe(), "-s", serial, "push", local, remote).CombinedOutput()
	if err != nil {
		return fmt.Errorf("adb push: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Pull copies a remote path to the local filesystem.
func Pull(serial, remote, local string) error {
	out, err := exec.Command(exe(), "-s", serial, "pull", remote, local).CombinedOutput()
	if err != nil {
		return fmt.Errorf("adb pull: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Watch polls the device list and emits the set of online serials whenever it
// changes. Polling (rather than `adb track-devices`) keeps disconnect
// detection simple and robust across adb versions.
func Watch(ctx context.Context, interval time.Duration, emit func([]string)) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	var last string
	tick := func() {
		devs, err := List()
		if err != nil {
			return
		}
		var serials []string
		for _, d := range online(devs) {
			serials = append(serials, d.Serial)
		}
		sort.Strings(serials)
		key := strings.Join(serials, ",")
		if key != last {
			last = key
			emit(serials)
		}
	}
	tick()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tick()
		}
	}
}
