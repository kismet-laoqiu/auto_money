package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"quantlab/internal/market"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/trader"
)

type RuntimeStore interface {
	AppendEvent(ctx context.Context, source string, evt sqlitepkg.LogEvent, raw []byte) (int64, error)
	ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]sqlitepkg.EventEnvelope, error)
	SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error
	LoadConsumerCursor(ctx context.Context, consumer string) (int64, error)
}

type NotificationSink interface {
	Notify(ctx context.Context, notification Notification) error
}

type RuntimeConfig struct {
	ConsumerKey      string
	DeadLetterSource string
	BatchSize        int
	MaxAttempts      int
	PollInterval     time.Duration
	Sources          []string
	DailySummary     DailySummarySchedule
	Store            RuntimeStore
	Service          NotificationSink
}

type Runtime struct {
	cfg    RuntimeConfig
	cursor int64
	loaded bool
}

type RuntimeHealth struct {
	ConsumerKey      string               `json:"consumer_key"`
	Cursor           int64                `json:"cursor"`
	Loaded           bool                 `json:"loaded"`
	Sources          []string             `json:"sources"`
	MaxAttempts      int                  `json:"max_attempts"`
	DeadLetterSource string               `json:"dead_letter_source"`
	DailySummary     DailySummarySchedule `json:"daily_summary"`
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	if cfg.ConsumerKey == "" {
		cfg.ConsumerKey = "notifierd"
	}
	if cfg.DeadLetterSource == "" {
		cfg.DeadLetterSource = "notifier"
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 128
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if len(cfg.Sources) == 0 {
		cfg.Sources = []string{"market.private", "trader", "platform"}
	}
	if cfg.DailySummary.LocationName == "" {
		cfg.DailySummary = DefaultDailySummarySchedule()
	}
	return &Runtime{cfg: cfg}
}

func (runtime *Runtime) Health() RuntimeHealth {
	return RuntimeHealth{
		ConsumerKey:      runtime.cfg.ConsumerKey,
		Cursor:           runtime.cursor,
		Loaded:           runtime.loaded,
		Sources:          append([]string(nil), runtime.cfg.Sources...),
		MaxAttempts:      runtime.cfg.MaxAttempts,
		DeadLetterSource: runtime.cfg.DeadLetterSource,
		DailySummary:     runtime.cfg.DailySummary,
	}
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
	if err := runtime.ensureLoaded(ctx); err != nil {
		return err
	}
	events, err := runtime.cfg.Store.ListEventsAfter(ctx, runtime.cursor, runtime.cfg.BatchSize, runtime.cfg.Sources...)
	if err != nil {
		return err
	}
	for _, event := range events {
		notification, ok, err := decodeNotificationEvent(event)
		if err != nil {
			return err
		}
		if ok && runtime.cfg.Service != nil {
			if err := runtime.deliverNotification(ctx, event, notification); err != nil {
				return err
			}
		}
		runtime.cursor = event.Seq
		if err := runtime.cfg.Store.SaveConsumerCursor(ctx, runtime.cfg.ConsumerKey, runtime.cursor); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *Runtime) deliverNotification(ctx context.Context, event sqlitepkg.EventEnvelope, notification Notification) error {
	var lastErr error
	for attempt := 1; attempt <= runtime.cfg.MaxAttempts; attempt++ {
		if err := runtime.cfg.Service.Notify(ctx, notification); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return runtime.appendDeadLetter(ctx, event, notification, lastErr)
}

func (runtime *Runtime) appendDeadLetter(ctx context.Context, event sqlitepkg.EventEnvelope, notification Notification, cause error) error {
	if runtime.cfg.Store == nil {
		return cause
	}
	deadLetter := DeadLetterEvent{
		EventIDValue:      fmt.Sprintf("dead-letter:%s:%d", event.EventID, event.Seq),
		SymbolValue:       event.Symbol,
		Ts:                time.Now().UTC(),
		NotificationKind:  notification.Kind,
		Title:             notification.Title,
		Summary:           notification.Summary,
		Details:           append([]string(nil), notification.Details...),
		Error:             cause.Error(),
		AttemptCount:      runtime.cfg.MaxAttempts,
		OriginalEventID:   event.EventID,
		OriginalEventKind: event.Kind,
	}
	body, err := json.Marshal(deadLetter)
	if err != nil {
		return err
	}
	_, err = runtime.cfg.Store.AppendEvent(ctx, runtime.cfg.DeadLetterSource, deadLetter, body)
	return err
}

func (runtime *Runtime) ensureLoaded(ctx context.Context) error {
	if runtime.loaded {
		return nil
	}
	if runtime.cfg.Store == nil {
		return fmt.Errorf("notifier runtime store is nil")
	}
	cursor, err := runtime.cfg.Store.LoadConsumerCursor(ctx, runtime.cfg.ConsumerKey)
	if err != nil {
		return err
	}
	runtime.cursor = cursor
	runtime.loaded = true
	return nil
}

func decodeNotificationEvent(event sqlitepkg.EventEnvelope) (Notification, bool, error) {
	switch event.Kind {
	case "order_fill":
		var payload market.OrderEvent
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return Notification{}, false, err
		}
		return Notification{
			Kind:    KindFill,
			Title:   fmt.Sprintf("%s fill", payload.SymbolValue),
			Summary: fmt.Sprintf("orderId=%s status=%s size=%g price=%g", payload.OrderID, payload.Status, payload.Size, payload.Price),
		}, true, nil
	case "risk.state_changed":
		var payload trader.RiskEvent
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return Notification{}, false, err
		}
		if payload.To != trader.ArmingDegraded && payload.To != trader.ArmingHalted {
			return Notification{}, false, nil
		}
		return Notification{
			Kind:    KindRiskHalt,
			Title:   fmt.Sprintf("%s risk state changed", payload.SymbolValue),
			Summary: fmt.Sprintf("from=%s to=%s reason=%s", payload.From, payload.To, payload.Reason),
		}, true, nil
	case "promotion.approved":
		return decodeGenericNotification(KindPromotionApproved, event.Payload)
	case "promotion.canary_degraded":
		return decodeGenericNotification(KindCanaryDegraded, event.Payload)
	case "insight.alert":
		return decodeInsightNotification(event.Payload)
	default:
		return Notification{}, false, nil
	}
}

func decodeGenericNotification(kind Kind, payload []byte) (Notification, bool, error) {
	var body struct {
		Title   string   `json:"title"`
		Summary string   `json:"summary"`
		Details []string `json:"details"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return Notification{}, false, err
	}
	return Notification{
		Kind:    kind,
		Title:   body.Title,
		Summary: body.Summary,
		Details: append([]string(nil), body.Details...),
	}, true, nil
}

func decodeInsightNotification(payload []byte) (Notification, bool, error) {
	var body struct {
		AlertType string   `json:"alert_type"`
		Title     string   `json:"title"`
		Summary   string   `json:"summary"`
		Details   []string `json:"details"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return Notification{}, false, err
	}
	kind := KindMarketAlert
	if body.AlertType == "daily_signal" {
		kind = KindDailySignal
	}
	return Notification{
		Kind:    kind,
		Title:   body.Title,
		Summary: body.Summary,
		Details: append([]string(nil), body.Details...),
	}, true, nil
}

type DeadLetterEvent struct {
	EventIDValue      string    `json:"event_id"`
	SymbolValue       string    `json:"symbol"`
	Ts                time.Time `json:"ts"`
	NotificationKind  Kind      `json:"notification_kind"`
	Title             string    `json:"title"`
	Summary           string    `json:"summary"`
	Details           []string  `json:"details"`
	Error             string    `json:"error"`
	AttemptCount      int       `json:"attempt_count"`
	OriginalEventID   string    `json:"original_event_id"`
	OriginalEventKind string    `json:"original_event_kind"`
}

func (event DeadLetterEvent) EventID() string      { return event.EventIDValue }
func (event DeadLetterEvent) Symbol() string       { return event.SymbolValue }
func (event DeadLetterEvent) EventTime() time.Time { return event.Ts }
func (event DeadLetterEvent) Kind() string         { return "notification.dead_letter" }
