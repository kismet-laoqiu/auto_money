package market

import "context"

type PublicFeed interface {
	Decode(raw []byte) ([]MarketEvent, error)
}

type PublicDecoder func(raw []byte) ([]MarketEvent, error)

func (decoder PublicDecoder) Decode(raw []byte) ([]MarketEvent, error) {
	return decoder(raw)
}

type EventSource interface {
	Events(ctx context.Context) <-chan []byte
}

type StreamFeed struct {
	source     EventSource
	decoder    PublicFeed
	aggregator *MicroBarAggregator
}

func NewPublicFeed(source EventSource, decoder PublicFeed, aggregator *MicroBarAggregator) *StreamFeed {
	return &StreamFeed{source: source, decoder: decoder, aggregator: aggregator}
}

func (feed *StreamFeed) Events(ctx context.Context) <-chan MarketEvent {
	out := make(chan MarketEvent)
	go func() {
		defer close(out)
		if feed.source == nil {
			<-ctx.Done()
			return
		}
		rawCh := feed.source.Events(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-rawCh:
				if !ok {
					return
				}
				events, err := feed.decoder.Decode(raw)
				if err != nil {
					continue
				}
				for _, event := range events {
					out <- event
					tick, ok := event.(TradeTickEvent)
					if !ok || feed.aggregator == nil {
						continue
					}
					for _, micro := range feed.aggregator.Push(tick) {
						out <- micro
					}
				}
			}
		}
	}()
	return out
}
