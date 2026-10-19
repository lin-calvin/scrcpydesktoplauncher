package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const AppName = "scrcpydesktoplauncher"

// ScrcpyOptions holds the per-launch scrcpy settings.
type ScrcpyOptions struct {
	Size               string  `json:"size"`        // "" or "WxH" (logical base size when hidpi_scale > 1)
	DPI                string  `json:"dpi"`         // "" or number
	BitRate            string  `json:"bit_rate"`    // "" or e.g. "16M", "8000000"
	HiDPIScale         float64 `json:"hidpi_scale"` // 1 = normal; >1 renders at physical pixels (disables flex)
	FlexDisplay        bool    `json:"flex_display"`
	NoVDDecorations    bool    `json:"no_vd_decorations"`
	KeepContentOnClose bool    `json:"keep_content_on_close"`
}

// Config is the persisted user configuration.
type Config struct {
	FolderName      string        `json:"folder_name"`
	NamePostfix     string        `json:"name_postfix"`     // e.g. " ({deviceid})"; {serial}/{deviceid} are substituted
	Scope           string        `json:"scope"`            // launchable | third_party
	Include         []string      `json:"include"`          // extra packages to force-include
	Exclude         []string      `json:"exclude"`          // packages to drop
	PreferTransport string        `json:"prefer_transport"` // usb | tcp | auto
	Audio           bool          `json:"audio"`
	Device          string        `json:"device"` // auto | <serial>
	SettingsEntry   bool          `json:"settings_entry"`
	Scrcpy          ScrcpyOptions `json:"scrcpy"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{
		FolderName:      "Android",
		Scope:           "launchable",
		PreferTransport: "auto",
		Audio:           true,
		Device:          "auto",
		SettingsEntry:   true,
		Scrcpy: ScrcpyOptions{
			HiDPIScale:         1,
			FlexDisplay:        true,
			KeepContentOnClose: false,
		},
	}
}

func configDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, AppName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", AppName)
}

// Path returns the config file path.
func Path() string { return filepath.Join(configDir(), "config.json") }

// Load reads the config, returning defaults if none exists.
func Load() (Config, error) {
	cfg := Default()
	b, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Default(), err
	}
	return cfg, nil
}

// Save writes the config atomically.
func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "config.json.tmp")
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, Path())
}
