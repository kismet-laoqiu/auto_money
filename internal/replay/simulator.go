package replay

import "quantlab/internal/trader"

type Summary struct {
	CommandsApplied   int `json:"commands_applied"`
	CandidatesApplied int `json:"candidates_applied"`
}

type Simulator interface {
	Apply(cmd trader.Command) error
	Snapshot() Summary
}

type LongOnlySimulator struct {
	summary Summary
}

func NewLongOnlySimulator() *LongOnlySimulator {
	return &LongOnlySimulator{}
}

func (sim *LongOnlySimulator) Apply(cmd trader.Command) error {
	sim.summary.CommandsApplied++
	if _, ok := cmd.(trader.Candidate); ok {
		sim.summary.CandidatesApplied++
	}
	return nil
}

func (sim *LongOnlySimulator) Snapshot() Summary {
	return sim.summary
}
