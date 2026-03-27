package market

type PublicFeed interface {
	Decode(raw []byte) ([]MarketEvent, error)
}

type PublicDecoder func(raw []byte) ([]MarketEvent, error)

func (decoder PublicDecoder) Decode(raw []byte) ([]MarketEvent, error) {
	return decoder(raw)
}
