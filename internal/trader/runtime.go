package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"quantlab/internal/market"
)

type RuntimeConfig struct {
	ConsumerKey   string
	CheckpointKey string
	OutputSource  string
	Sources       []string
	BatchSize     int
	PollInterval  time.Duration
	ObserveOnly   bool
	RunID         string
	MaxLeverage   int
	Store         RuntimeStore
	Engine        *Engine
	Risk          *RiskEngine
	Reconciler    *Reconciler
}

type Runtime struct {
	cfg       RuntimeConfig
	engine    *Engine
	cursor    int64
	loaded    bool
	positions map[string]SymbolPosition
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	if cfg.ConsumerKey == "" {
		cfg.ConsumerKey = "traderd"
	}
	if cfg.CheckpointKey == "" {
		cfg.CheckpointKey = "trader.runtime"
	}
	if cfg.OutputSource == "" {
		cfg.OutputSource = "trader"
	}
	if len(cfg.Sources) == 0 {
		cfg.Sources = []string{"market.bootstrap", "market.public", "market.private"}
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 256
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.Engine == nil {
		cfg.Engine = NewEngine(Config{})
	}
	if cfg.MaxLeverage <= 0 {
		cfg.MaxLeverage = 3
	}
	return &Runtime{
		cfg:       cfg,
		engine:    cfg.Engine,
		positions: map[string]SymbolPosition{},
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
		if err := runtime.cfg.Store.SaveCheckpoint(ctx, runtime.cfg.CheckpointKey, runtime.engine.State()); err != nil {
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
		return fmt.Errorf("trader runtime store is nil")
	}
	cursor, err := runtime.cfg.Store.LoadConsumerCursor(ctx, runtime.cfg.ConsumerKey)
	if err != nil {
		return err
	}
	state, err := runtime.cfg.Store.LoadCheckpoint(ctx, runtime.cfg.CheckpointKey)
	if err != nil {
		return err
	}
	if state.ArmingState != "" || len(state.Symbols) != 0 {
		runtime.engine.Restore(state)
	}
	runtime.cursor = cursor
	runtime.loaded = true
	return nil
}

func (runtime *Runtime) handleEnvelope(ctx context.Context, env EventEnvelope) error {
	event, err := DecodeMarketEvent(env)
	if err != nil {
		return err
	}
	if event == nil {
		return nil
	}
	switch value := event.(type) {
	case market.PositionEvent:
		return runtime.handlePositionEvent(ctx, value)
	case market.OrderEvent:
		return nil
	}
	commands, err := runtime.engine.Advance(event)
	if err != nil {
		return err
	}
	for _, command := range commands {
		if err := runtime.emitCommand(ctx, command); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *Runtime) emitCommand(ctx context.Context, command Command) error {
	switch value := command.(type) {
	case Candidate:
		if err := runtime.appendEvent(ctx, BuildCandidateEvent(value)); err != nil {
			return err
		}
		return runtime.maybeEmitEntryIntent(ctx, value)
	default:
		return nil
	}
}

func (runtime *Runtime) maybeEmitEntryIntent(ctx context.Context, candidate Candidate) error {
	if runtime.cfg.ObserveOnly {
		return nil
	}
	if runtime.engine.State().ArmingState != ArmingArmed {
		return nil
	}
	intent := BuildEntryIntentEvent(runtime.cfg.RunID, runtime.cfg.MaxLeverage, candidate)
	if runtime.cfg.Risk != nil {
		runtime.cfg.Risk.UpdateState(runtime.engine.State())
		verdict := runtime.cfg.Risk.Check(EntryIntent{
			Symbol:   candidate.Symbol,
			Notional: candidate.Entry * parseIntentSize(intent.Size),
		})
		if !verdict.Allow {
			return nil
		}
	}
	if err := runtime.appendEvent(ctx, intent); err != nil {
		return err
	}
	runtime.positions[candidate.Symbol] = SymbolPosition{
		Symbol:      candidate.Symbol,
		Qty:         signedIntentQty(intent),
		ProductType: intent.ProductType,
		MarginMode:  intent.MarginMode,
		MarginCoin:  intent.MarginCoin,
	}
	return nil
}

func (runtime *Runtime) handlePositionEvent(ctx context.Context, event market.PositionEvent) error {
	if runtime.cfg.Reconciler == nil {
		return nil
	}
	local := PositionSnapshot{Symbol: event.SymbolValue}
	if position, ok := runtime.positions[event.SymbolValue]; ok {
		local.Qty = position.Qty
	}
	verdict := runtime.cfg.Reconciler.Compare(local, PositionSnapshot{Symbol: event.SymbolValue, Qty: event.Qty})
	return runtime.applyReconcileVerdict(ctx, event, verdict)
}

func (runtime *Runtime) applyReconcileVerdict(ctx context.Context, event market.PositionEvent, verdict ReconcileVerdict) error {
	if !shouldDowngrade(runtime.engine.State().ArmingState, verdict.NextArmingState) {
		return nil
	}
	state := runtime.engine.State()
	from := state.ArmingState
	state.ArmingState = verdict.NextArmingState
		 runtime.engine.Restore(state)
	return runtime.appendEvent(ctx, RiskEvent{
		EventIDValue: fmt.Sprintf("risk:%s:%s:%d", event.SymbolValue, verdict.Reason, event.Ts.UnixNano()),
		SymbolValue:  event.SymbolValue,
		Ts:           event.Ts,
		From:         from,
		To:           verdict.NextArmingState,
		Reason:       verdict.Reason,
	})
}

func (runtime *Runtime) appendEvent(ctx context.Context, event EventLog) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = runtime.cfg.Store.AppendEvent(ctx, runtime.cfg.OutputSource, event, body)
	if isDuplicateEventErr(err) {
		return nil
	}
	return err
}

func isDuplicateEventErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "event_log.event_id")
}

func shouldDowngrade(current, next ArmingState) bool {
	if next != ArmingDegraded && next != ArmingHalted {
		return false
	}
	rank := map[ArmingState]int{
		ArmingBooting:  0,
		ArmingSafe:     1,
		ArmingArmed:    2,
		ArmingDegraded: 3,
		ArmingHalted:   4,
	}
	return rank[next] > rank[current]
}
