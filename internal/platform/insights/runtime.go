package insights

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	sqlitepkg "quantlab/internal/store/sqlite"
)

type EventStore interface {
	AppendEvent(ctx context.Context, source string, evt sqlitepkg.LogEvent, raw []byte) (int64, error)
}

type Detector interface {
	DetectAlerts(ctx context.Context) ([]Event, error)
}

type RuntimeConfig struct {
	Service      Detector
	Store        EventStore
	Source       string
	PollInterval time.Duration
}

type Runtime struct {
	cfg RuntimeConfig
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	if cfg.Source == "" {
		cfg.Source = "platform"
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Minute
	}
	return &Runtime{cfg: cfg}
}

func (runtime *Runtime) Run(ctx context.Context) error {
	ticker := time.NewTicker(runtime.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := runtime.ProcessAvailable(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (runtime *Runtime) ProcessAvailable(ctx context.Context) error {
	if runtime == nil || runtime.cfg.Service == nil || runtime.cfg.Store == nil {
		return nil
	}
	alerts, err := runtime.cfg.Service.DetectAlerts(ctx)
	if err != nil {
		return err
	}
	for _, alert := range alerts {
		body, err := json.Marshal(alert)
		if err != nil {
			return err
		}
		if _, err := runtime.cfg.Store.AppendEvent(ctx, runtime.cfg.Source, alert, body); err != nil {
			if isDuplicateEventErr(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func isDuplicateEventErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed: event_log.event_id")
}
