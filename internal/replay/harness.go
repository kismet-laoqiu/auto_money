package replay

import (
	"encoding/json"
	"os"
	"path/filepath"

	"quantlab/internal/market"
	"quantlab/internal/trader"
)

type Engine interface {
	Advance(evt market.MarketEvent) ([]trader.Command, error)
	State() trader.EngineState
}

type Report struct {
	EventCount     int                `json:"event_count"`
	CommandCount   int                `json:"command_count"`
	CandidateCount int                `json:"candidate_count"`
	FinalState     trader.EngineState `json:"final_state"`
	Simulation     Summary            `json:"simulation"`
}

type Harness struct {
	engine    Engine
	simulator Simulator
}

func NewHarness(engine Engine, simulator Simulator) *Harness {
	return &Harness{engine: engine, simulator: simulator}
}

func (h *Harness) Run(events []market.MarketEvent) (Report, error) {
	report := Report{EventCount: len(events)}
	for _, event := range events {
		commands, err := h.engine.Advance(event)
		if err != nil {
			return Report{}, err
		}
		report.CommandCount += len(commands)
		for _, cmd := range commands {
			if _, ok := cmd.(trader.Candidate); ok {
				report.CandidateCount++
			}
			if h.simulator != nil {
				if err := h.simulator.Apply(cmd); err != nil {
					return Report{}, err
				}
			}
		}
	}
	report.FinalState = h.engine.State()
	if h.simulator != nil {
		report.Simulation = h.simulator.Snapshot()
	}
	return report, nil
}

func (h *Harness) RunAndWrite(events []market.MarketEvent, artifactDir string) (Report, error) {
	report, err := h.Run(events)
	if err != nil {
		return Report{}, err
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return Report{}, err
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return Report{}, err
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "replay-report.json"), append(body, '\n'), 0o644); err != nil {
		return Report{}, err
	}
	return report, nil
}
