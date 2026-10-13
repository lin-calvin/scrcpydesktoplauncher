package notify

import (
	"os/exec"
	"strings"
)

// Send shows a desktop notification (best effort, never fails the caller).
func Send(summary, body string) {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return
	}
	_ = exec.Command("notify-send", "-a", "Android Launcher", summary, body).Run()
}

// Error shows an error notification.
func Error(summary string, err error) {
	if err == nil {
		return
	}
	Send(summary, strings.TrimSpace(err.Error()))
}
