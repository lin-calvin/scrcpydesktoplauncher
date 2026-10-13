package daemon

import (
	"context"
	"log"
	"time"

	"scrcpydesktoplauncher/internal/adb"
	"scrcpydesktoplauncher/internal/config"
	"scrcpydesktoplauncher/internal/reconcile"
)

const (
	pollInterval   = 2 * time.Second
	periodicResync = 5 * time.Minute
)

// Run watches adb for device changes and reconciles the launcher entries.
// It blocks until ctx is cancelled.
func Run(ctx context.Context) error {
	if err := adb.StartServer(); err != nil {
		log.Printf("adb start-server: %v", err)
	}

	mu := make(chan struct{}, 1)
	mu <- struct{}{}
	reconcileNow := func(reason string) {
		<-mu
		defer func() { mu <- struct{}{} }()
		cfg, err := config.Load()
		if err != nil {
			log.Printf("config: %v", err)
			return
		}
		res, err := reconcile.Run(cfg)
		if err != nil {
			if err == adb.ErrNoDevice {
				return
			}
			log.Printf("reconcile (%s): %v", reason, err)
			return
		}
		log.Printf("reconcile (%s): serial=%s total=%d added=%d removed=%d updated=%d",
			reason, res.Serial, res.Total, len(res.Added), len(res.Removed), res.Updated)
	}

	log.Printf("daemon started; polling adb every %s", pollInterval)
	go adb.Watch(ctx, pollInterval, func(serials []string) {
		if len(serials) == 0 {
			log.Printf("no device connected")
			return
		}
		log.Printf("device(s): %v", serials)
		reconcileNow("connect")
	})

	// Periodic resync catches installs/uninstalls while a device stays attached.
	t := time.NewTicker(periodicResync)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if devs, err := adb.List(); err == nil {
				for _, d := range devs {
					if d.State == "device" {
						reconcileNow("periodic")
						break
					}
				}
			}
		}
	}
}
