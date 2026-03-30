package promotion

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"quantlab/internal/backtest"
	sqlitepkg "quantlab/internal/store/sqlite"
)

type State string

const (
	StateBacktestPassed State = "backtest_passed"
	StateShadowRunning  State = "shadow_running"
	StateShadowPassed   State = "shadow_passed"
	StateCanaryRunning  State = "canary_running"
	StateCanaryDegraded State = "canary_degraded"
	StateLiveActive     State = "live_active"
	StateRolledBack     State = "rolled_back"
)

type Action string

const (
	ActionRequest       Action = "request"
	ActionStartShadow   Action = "start_shadow"
	ActionPassShadow    Action = "pass_shadow"
	ActionStartCanary   Action = "start_canary"
	ActionDegradeCanary Action = "degrade_canary"
	ActionApprove       Action = "approve"
	ActionRollback      Action = "rollback"
)

type Request struct {
	ID             string    `json:"id"`
	StrategyID     string    `json:"strategy_id"`
	Version        string    `json:"version"`
	ConfigPath     string    `json:"config_path"`
	State          State     `json:"state"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ObjectiveScore float64   `json:"objective_score"`
	FinalScore     float64   `json:"final_score"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	Details        []string  `json:"details"`
}

type CreateInput struct {
	StrategyID string
	Version    string
	ConfigPath string
	Refresh    bool
}

type Event struct {
	EventIDValue   string    `json:"event_id"`
	Ts             time.Time `json:"ts"`
	Action         Action    `json:"action"`
	PromotionID    string    `json:"promotion_id"`
	StrategyID     string    `json:"strategy_id"`
	Version        string    `json:"version"`
	ConfigPath     string    `json:"config_path"`
	State          State     `json:"state"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ObjectiveScore float64   `json:"objective_score"`
	FinalScore     float64   `json:"final_score"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	Details        []string  `json:"details"`
}

func (event Event) EventID() string      { return event.EventIDValue }
func (event Event) Symbol() string       { return event.StrategyID }
func (event Event) EventTime() time.Time { return event.Ts }

func (event Event) Kind() string {
	switch event.Action {
	case ActionRequest:
		return "promotion.requested"
	case ActionStartShadow:
		return "promotion.shadow_started"
	case ActionPassShadow:
		return "promotion.shadow_passed"
	case ActionStartCanary:
		return "promotion.canary_started"
	case ActionDegradeCanary:
		return "promotion.canary_degraded"
	case ActionApprove:
		return "promotion.approved"
	case ActionRollback:
		return "promotion.rolled_back"
	default:
		return "promotion.unknown"
	}
}

func (event Event) Request() Request {
	return Request{
		ID:             event.PromotionID,
		StrategyID:     event.StrategyID,
		Version:        event.Version,
		ConfigPath:     event.ConfigPath,
		State:          event.State,
		CreatedAt:      event.CreatedAt,
		UpdatedAt:      event.UpdatedAt,
		ObjectiveScore: event.ObjectiveScore,
		FinalScore:     event.FinalScore,
		Title:          event.Title,
		Summary:        event.Summary,
		Details:        append([]string(nil), event.Details...),
	}
}

func Advance(current State, action Action) (State, error) {
	switch action {
	case ActionStartShadow:
		if current != StateBacktestPassed {
			return "", fmt.Errorf("cannot start shadow from %s", current)
		}
		return StateShadowRunning, nil
	case ActionPassShadow:
		if current != StateShadowRunning {
			return "", fmt.Errorf("cannot pass shadow from %s", current)
		}
		return StateShadowPassed, nil
	case ActionStartCanary:
		if current != StateShadowPassed {
			return "", fmt.Errorf("cannot start canary from %s", current)
		}
		return StateCanaryRunning, nil
	case ActionDegradeCanary:
		if current != StateCanaryRunning {
			return "", fmt.Errorf("cannot degrade canary from %s", current)
		}
		return StateCanaryDegraded, nil
	case ActionApprove:
		if current != StateCanaryRunning {
			return "", fmt.Errorf("cannot approve from %s", current)
		}
		return StateLiveActive, nil
	case ActionRollback:
		if current != StateCanaryRunning && current != StateCanaryDegraded && current != StateLiveActive {
			return "", fmt.Errorf("cannot rollback from %s", current)
		}
		return StateRolledBack, nil
	default:
		return "", fmt.Errorf("unsupported action %s", action)
	}
}

type BacktestRunner interface {
	RunConfig(ctx context.Context, configPath string, refresh bool) (backtest.Result, error)
}

type Store interface {
	AppendEvent(ctx context.Context, source string, evt sqlitepkg.LogEvent, raw []byte) (int64, error)
	ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]sqlitepkg.EventEnvelope, error)
}

type Config struct {
	Store     Store
	Backtests BacktestRunner
	Source    string
	Now       func() time.Time
}

type Service struct {
	cfg Config
}

func NewService(cfg Config) *Service {
	if cfg.Source == "" {
		cfg.Source = "platform"
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{cfg: cfg}
}

func (service *Service) CreateRequest(ctx context.Context, input CreateInput) (Request, error) {
	if service.cfg.Store == nil {
		return Request{}, fmt.Errorf("promotion store is nil")
	}
	if service.cfg.Backtests == nil {
		return Request{}, fmt.Errorf("promotion backtest runner is nil")
	}
	if input.StrategyID == "" || input.Version == "" || input.ConfigPath == "" {
		return Request{}, fmt.Errorf("strategy_id, version, and config_path are required")
	}
	result, err := service.cfg.Backtests.RunConfig(ctx, input.ConfigPath, input.Refresh)
	if err != nil {
		return Request{}, err
	}
	now := service.cfg.Now().UTC()
	request := Request{
		ID:             buildPromotionID(input.StrategyID, input.Version, now),
		StrategyID:     input.StrategyID,
		Version:        input.Version,
		ConfigPath:     input.ConfigPath,
		State:          StateBacktestPassed,
		CreatedAt:      now,
		UpdatedAt:      now,
		ObjectiveScore: result.ObjectiveScore,
		FinalScore:     result.FinalScore,
		Title:          fmt.Sprintf("%s %s backtest passed", input.StrategyID, input.Version),
		Summary:        fmt.Sprintf("objective=%.4f final=%.4f", result.ObjectiveScore, result.FinalScore),
		Details:        []string{fmt.Sprintf("config=%s", input.ConfigPath)},
	}
	return request, service.append(ctx, ActionRequest, request, "")
}

func (service *Service) Get(ctx context.Context, id string) (Request, error) {
	requests, err := service.load(ctx)
	if err != nil {
		return Request{}, err
	}
	request, ok := requests[id]
	if !ok {
		return Request{}, fmt.Errorf("promotion %s not found", id)
	}
	return request, nil
}

func (service *Service) List(ctx context.Context) ([]Request, error) {
	requests, err := service.load(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Request, 0, len(requests))
	for _, request := range requests {
		out = append(out, request)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func (service *Service) StartShadow(ctx context.Context, id string) (Request, error) {
	return service.transition(ctx, id, ActionStartShadow, "")
}

func (service *Service) PassShadow(ctx context.Context, id string) (Request, error) {
	return service.transition(ctx, id, ActionPassShadow, "")
}

func (service *Service) StartCanary(ctx context.Context, id string) (Request, error) {
	return service.transition(ctx, id, ActionStartCanary, "")
}

func (service *Service) DegradeCanary(ctx context.Context, id, reason string) (Request, error) {
	return service.transition(ctx, id, ActionDegradeCanary, reason)
}

func (service *Service) Approve(ctx context.Context, id string) (Request, error) {
	return service.transition(ctx, id, ActionApprove, "")
}

func (service *Service) Rollback(ctx context.Context, id, reason string) (Request, error) {
	return service.transition(ctx, id, ActionRollback, reason)
}

func (service *Service) transition(ctx context.Context, id string, action Action, reason string) (Request, error) {
	request, err := service.Get(ctx, id)
	if err != nil {
		return Request{}, err
	}
	request, err = transitionRequest(action, request, service.cfg.Now(), reason)
	if err != nil {
		return Request{}, err
	}
	return request, service.append(ctx, action, request, reason)
}

func (service *Service) append(ctx context.Context, action Action, request Request, reason string) error {
	event := Event{
		EventIDValue:   fmt.Sprintf("promotion:%s:%s:%d", request.ID, action, request.UpdatedAt.UnixNano()),
		Ts:             request.UpdatedAt,
		Action:         action,
		PromotionID:    request.ID,
		StrategyID:     request.StrategyID,
		Version:        request.Version,
		ConfigPath:     request.ConfigPath,
		State:          request.State,
		CreatedAt:      request.CreatedAt,
		UpdatedAt:      request.UpdatedAt,
		ObjectiveScore: request.ObjectiveScore,
		FinalScore:     request.FinalScore,
		Title:          request.Title,
		Summary:        request.Summary,
		Details:        append([]string(nil), request.Details...),
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = service.cfg.Store.AppendEvent(ctx, service.cfg.Source, event, body)
	return err
}

func (service *Service) load(ctx context.Context) (map[string]Request, error) {
	if service.cfg.Store == nil {
		return nil, fmt.Errorf("promotion store is nil")
	}
	out := map[string]Request{}
	afterSeq := int64(0)
	for {
		events, err := service.cfg.Store.ListEventsAfter(ctx, afterSeq, 256, service.cfg.Source)
		if err != nil {
			return nil, err
		}
		if len(events) == 0 {
			return out, nil
		}
		for _, envelope := range events {
			afterSeq = envelope.Seq
			if !strings.HasPrefix(envelope.Kind, "promotion.") {
				continue
			}
			var event Event
			if err := json.Unmarshal(envelope.Payload, &event); err != nil {
				return nil, err
			}
			out[event.PromotionID] = event.Request()
		}
	}
}

func renderTransition(action Action, request Request, reason string) (string, string, []string) {
	details := []string{
		fmt.Sprintf("promotion_id=%s", request.ID),
		fmt.Sprintf("strategy=%s", request.StrategyID),
		fmt.Sprintf("version=%s", request.Version),
	}
	if request.ConfigPath != "" {
		details = append(details, fmt.Sprintf("config=%s", request.ConfigPath))
	}
	if reason != "" {
		details = append(details, fmt.Sprintf("reason=%s", reason))
	}
	switch action {
	case ActionStartShadow:
		return fmt.Sprintf("%s %s shadow started", request.StrategyID, request.Version), "shadow gate running", details
	case ActionPassShadow:
		return fmt.Sprintf("%s %s shadow passed", request.StrategyID, request.Version), "shadow gate passed", details
	case ActionStartCanary:
		return fmt.Sprintf("%s %s canary started", request.StrategyID, request.Version), "canary gate running", details
	case ActionDegradeCanary:
		return fmt.Sprintf("%s %s canary degraded", request.StrategyID, request.Version), "canary gate degraded", details
	case ActionApprove:
		return fmt.Sprintf("%s %s promotion approved", request.StrategyID, request.Version), "promotion is live_active", details
	case ActionRollback:
		return fmt.Sprintf("%s %s rolled back", request.StrategyID, request.Version), "promotion rolled back", details
	default:
		return request.Title, request.Summary, details
	}
}

func buildPromotionID(strategyID, version string, now time.Time) string {
	return fmt.Sprintf("%s-%s-%d", slug(strategyID), slug(version), now.UnixNano())
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		default:
			builder.WriteByte('-')
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "promotion"
	}
	return out
}
