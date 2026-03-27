package market

import (
	"fmt"
	"time"
)

type MicroBarAggregator struct {
	interval time.Duration
	states   map[string]*microBarState
}

type microBarState struct {
	bucketStart time.Time
	open        float64
	high        float64
	low         float64
	close       float64
	volume      float64
}

func NewMicroBarAggregator(interval time.Duration) *MicroBarAggregator {
	if interval <= 0 {
		interval = time.Second
	}
	return &MicroBarAggregator{
		interval: interval,
		states:   map[string]*microBarState{},
	}
}

func (aggregator *MicroBarAggregator) Push(tick TradeTickEvent) []MarketEvent {
	bucketStart := tick.Ts.Truncate(aggregator.interval)
	state := aggregator.states[tick.SymbolValue]
	if state == nil {
		aggregator.states[tick.SymbolValue] = newMicroBarState(bucketStart, tick)
		return nil
	}
	if bucketStart.Before(state.bucketStart) {
		return nil
	}
	if bucketStart.Equal(state.bucketStart) {
		state.high = maxFloat(state.high, tick.Price)
		state.low = minFloat(state.low, tick.Price)
		state.close = tick.Price
		state.volume += tick.Size
		return nil
	}
	closed := MicroBarClosedEvent{
		EventIDValue: fmt.Sprintf("micro_bar:%s:%d", tick.SymbolValue, state.bucketStart.UnixMilli()),
		SymbolValue:  tick.SymbolValue,
		OpenedAt:     state.bucketStart,
		ClosedAt:     state.bucketStart.Add(aggregator.interval),
		Open:         state.open,
		High:         state.high,
		Low:          state.low,
		Close:        state.close,
		Volume:       state.volume,
	}
	aggregator.states[tick.SymbolValue] = newMicroBarState(bucketStart, tick)
	return []MarketEvent{closed}
}

func newMicroBarState(bucketStart time.Time, tick TradeTickEvent) *microBarState {
	return &microBarState{
		bucketStart: bucketStart,
		open:        tick.Price,
		high:        tick.Price,
		low:         tick.Price,
		close:       tick.Price,
		volume:      tick.Size,
	}
}

func maxFloat(left, right float64) float64 {
	if right > left {
		return right
	}
	return left
}

func minFloat(left, right float64) float64 {
	if right < left {
		return right
	}
	return left
}
