package common

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Mapping struct {
	Venue           string     `json:"venue"`
	MarketType      MarketType `json:"market_type"`
	Raw             string     `json:"raw"`
	CanonicalSymbol string     `json:"canonical_symbol"`
	VenueSymbol     string     `json:"venue_symbol,omitempty"`
	AssetID         string     `json:"asset_id,omitempty"`
}

type Mapper struct {
	mappings map[string]Instrument
}

func NewMapper(entries []Mapping) *Mapper {
	mappings := make(map[string]Instrument, len(entries))
	for _, entry := range entries {
		venue := normalizeVenue(entry.Venue)
		marketType := normalizeMarketType(entry.MarketType)
		raw := normalizeRaw(entry.Raw)
		if venue == "" || marketType == "" || raw == "" || strings.TrimSpace(entry.CanonicalSymbol) == "" {
			continue
		}
		venueSymbol := strings.TrimSpace(entry.VenueSymbol)
		if venueSymbol == "" {
			venueSymbol = strings.TrimSpace(entry.Raw)
		}
		mappings[mappingKey(venue, marketType, raw)] = Instrument{
			Venue:           venue,
			MarketType:      marketType,
			CanonicalSymbol: strings.ToUpper(strings.TrimSpace(entry.CanonicalSymbol)),
			VenueSymbol:     venueSymbol,
			AssetID:         strings.TrimSpace(entry.AssetID),
		}
	}
	return &Mapper{mappings: mappings}
}

func LoadMappings(path string) ([]Mapping, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []Mapping
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("decode mappings: %w", err)
	}
	return entries, nil
}

func (mapper *Mapper) Resolve(venue string, marketType MarketType, raw string) (Instrument, bool) {
	if mapper == nil {
		return Instrument{}, false
	}
	key := mappingKey(normalizeVenue(venue), normalizeMarketType(marketType), normalizeRaw(raw))
	instrument, ok := mapper.mappings[key]
	return instrument, ok
}

func normalizeVenue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeMarketType(value MarketType) MarketType {
	switch strings.ToLower(strings.TrimSpace(string(value))) {
	case "spot":
		return MarketTypeSpot
	default:
		return MarketTypePerp
	}
}

func normalizeRaw(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func mappingKey(venue string, marketType MarketType, raw string) string {
	return venue + "|" + strings.ToLower(string(marketType)) + "|" + raw
}
