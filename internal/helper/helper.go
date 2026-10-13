package helper

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scrcpydesktoplauncher/internal/adb"
	"scrcpydesktoplauncher/internal/xdg"
)

//go:embed helper.dex
var dex []byte

const (
	remoteDir = "/data/local/tmp/scrcpydesktoplauncher"
	remoteDex = remoteDir + "/alauncher.dex"
	mainClass = "com.calvin.alauncher.AppList"
)

// App is one launchable Android app.
type App struct {
	Package string `json:"package"`
	Label   string `json:"label"`
	System  bool   `json:"system"`
	Icon    string `json:"icon"` // basename of the PNG
}

// Dump runs the helper on the device and pulls manifest + icons into the cache.
// It returns the parsed apps and the local directory containing their icons.
func Dump(serial string) ([]App, string, error) {
	cache := filepath.Join(xdg.CacheDir(), sanitize(serial))
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return nil, "", err
	}

	dexLocal := filepath.Join(cache, "alauncher.dex")
	if err := os.WriteFile(dexLocal, dex, 0o644); err != nil {
		return nil, "", err
	}

	if _, err := adb.Shell(serial, fmt.Sprintf("mkdir -p %s", remoteDir)); err != nil {
		return nil, "", err
	}
	if err := adb.Push(serial, dexLocal, remoteDex); err != nil {
		return nil, "", err
	}
	if _, err := adb.Shell(serial, fmt.Sprintf("rm -rf %s/icons %s/manifest.jsonl", remoteDir, remoteDir)); err != nil {
		return nil, "", err
	}
	if out, err := adb.Shell(serial, fmt.Sprintf("CLASSPATH=%s app_process /system/bin %s %s", remoteDex, mainClass, remoteDir)); err != nil {
		return nil, "", fmt.Errorf("helper run failed: %w (%s)", err, strings.TrimSpace(out))
	}

	pullDir := filepath.Join(cache, "dump")
	_ = os.RemoveAll(pullDir)
	if err := adb.Pull(serial, remoteDir, pullDir); err != nil {
		return nil, "", err
	}

	apps, err := parseManifest(filepath.Join(pullDir, "manifest.jsonl"))
	if err != nil {
		return nil, "", err
	}
	return apps, filepath.Join(pullDir, "icons"), nil
}

func parseManifest(path string) ([]App, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var apps []App
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var a App
		if err := json.Unmarshal([]byte(line), &a); err != nil {
			continue
		}
		apps = append(apps, a)
	}
	return apps, sc.Err()
}

func sanitize(s string) string {
	r := strings.NewReplacer(":", "_", "/", "_")
	return r.Replace(s)
}
