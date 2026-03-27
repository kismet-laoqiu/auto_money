package trader

import (
	"time"

	"quantlab/internal/core"
)

type Command interface {
	Name() string
}

type Candidate struct {
	Symbol   string
	Interval string
	Ts       time.Time
	Side     core.Side
	Score    float64
	Entry    float64
	Stop     float64
	Target   float64
	Reasons  []string
}

func (candidate Candidate) Name() string {
	return "candidate"
}
