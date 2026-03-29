package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/trader"
)

type RuntimeStore interface {
	ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]trader.EventEnvelope, error)
	SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error
	LoadConsumerCursor(ctx context.Context, consumer string) (int64, error)
}

type RuntimeConfig struct {
	ConsumerKey string
	Sources     []string
	BatchSize   int
	PollInterval time.Duration
	ArtifactDir string
	MaxLeverage int
	Store       RuntimeStore
	Service     *Service
}

type Runtime struct {
	cfg    RuntimeConfig
	cursor int64
	loaded bool
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	if cfg.ConsumerKey == "" {
		cfg.ConsumerKey = "agentd"
	}
	if len(cfg.Sources) == 0 {
		cfg.Sources = []string{"trader"}
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 128
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.MaxLeverage == 0 {
		cfg.MaxLeverage = 3
	}
	if cfg.ArtifactDir == "" {
		cfg.ArtifactDir = filepath.Join("artifacts", "agentd")
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
	if err := runtime.ensureLoaded(ctx); err != nil {
		return err
	}
	if runtime.cfg.Store == nil {
		return fmt.Errorf("agent runtime store is nil")
	}
	if runtime.cfg.Service == nil {
		return fmt.Errorf("agent runtime service is nil")
	}
	envelopes, err := runtime.cfg.Store.ListEventsAfter(ctx, runtime.cursor, runtime.cfg.BatchSize, runtime.cfg.Sources...)
	if err != nil {
		return err
	}
	for _, env := range envelopes {
		if err := runtime.handleEnvelope(ctx, env); err != nil {
			return err
		}
		runtime.cursor = env.Seq
		if err := runtime.cfg.Store.SaveConsumerCursor(ctx, runtime.cfg.ConsumerKey, runtime.cursor); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *Runtime) ensureLoaded(ctx context.Context) error {
	if runtime.loaded {
		return nil
	}
	if runtime.cfg.Store == nil {
		return fmt.Errorf("agent runtime store is nil")
	}
	cursor, err := runtime.cfg.Store.LoadConsumerCursor(ctx, runtime.cfg.ConsumerKey)
	if err != nil {
		return err
	}
	runtime.cursor = cursor
	runtime.loaded = true
	return nil
}

func (runtime *Runtime) handleEnvelope(ctx context.Context, env trader.EventEnvelope) error {
	switch env.Kind {
	case "candidate.created":
		var event trader.CandidateEvent
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return err
		}
		resp, err := runtime.cfg.Service.ReviewCandidate(ctx, CandidatePacket{
			Symbol:          event.SymbolValue,
			SuggestedAction: suggestedActionForSide(event.Side),
			MaxLeverage:     runtime.cfg.MaxLeverage,
			Reasons:         append([]string(nil), event.Reasons...),
		})
		if err != nil {
			return err
		}
		return runtime.writeArtifact(env.Seq, env.Kind, map[string]any{"event": event, "response": resp, "output_text": resp.OutputText()})
	case "risk.state_changed":
		var event trader.RiskEvent
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return err
		}
		resp, err := runtime.cfg.Service.ExplainRisk(ctx, RiskPacket{
			Symbol: event.SymbolValue,
			From:   string(event.From),
			To:     string(event.To),
			Reason: event.Reason,
		})
		if err != nil {
			return err
		}
		return runtime.writeArtifact(env.Seq, env.Kind, map[string]any{"event": event, "response": resp, "output_text": resp.OutputText()})
	default:
		return nil
	}
}

func (runtime *Runtime) writeArtifact(seq int64, kind string, payload any) error {
	if err := os.MkdirAll(runtime.cfg.ArtifactDir, 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%06d-%s.json", seq, sanitizeArtifactName(kind))
	return os.WriteFile(filepath.Join(runtime.cfg.ArtifactDir, filename), append(body, '\n'), 0o644)
}

func sanitizeArtifactName(kind string) string {
	name := kind
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

func suggestedActionForSide(side core.Side) string {
	switch side {
	case core.Long:
		return "probe_long"
	case core.Short:
		return "probe_short"
	default:
		return "stand_aside"
	}
}
