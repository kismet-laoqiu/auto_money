package core

import (
	"math"
	"testing"
	"time"
)

func TestExtractWaveStructureFeaturesRisingSequence(t *testing.T) {
	bars := []Bar{
		testBar(0, 100, 101, 99, 100, 100),
		testBar(1, 100, 112, 104, 111, 100),
		testBar(2, 111, 106, 96, 103, 100),
		testBar(3, 103, 120, 104, 118, 100),
		testBar(4, 118, 111, 98, 109, 100),
		testBar(5, 109, 128, 110, 126, 100),
		testBar(6, 126, 119, 100, 117, 100),
		testBar(7, 117, 134, 118, 132, 100),
		testBar(8, 132, 128, 110, 126, 100),
	}

	got := ExtractWaveStructureFeatures(bars, len(bars)-1, 1, len(bars), 0.01)
	if !got.ValidStructure {
		t.Fatalf("expected valid structure")
	}
	if got.WaveUpScore <= got.WaveDownScore {
		t.Fatalf("expected bullish structure, got up=%.4f down=%.4f", got.WaveUpScore, got.WaveDownScore)
	}
	if got.ImpulseExtensionRatio <= 0 {
		t.Fatalf("expected positive extension ratio, got %.4f", got.ImpulseExtensionRatio)
	}
}

func TestExtractWaveStructureFeaturesFallingSequence(t *testing.T) {
	bars := []Bar{
		testBar(0, 130, 131, 120, 121, 100),
		testBar(1, 121, 133, 122, 132, 100),
		testBar(2, 132, 126, 118, 120, 100),
		testBar(3, 120, 129, 121, 128, 100),
		testBar(4, 128, 120, 112, 114, 100),
		testBar(5, 114, 122, 115, 121, 100),
		testBar(6, 121, 114, 106, 108, 100),
		testBar(7, 108, 116, 109, 115, 100),
		testBar(8, 115, 108, 98, 100, 100),
		testBar(9, 100, 111, 101, 110, 100),
	}

	got := ExtractWaveStructureFeatures(bars, len(bars)-1, 1, len(bars), 0.01)
	if !got.ValidStructure {
		t.Fatalf("expected valid structure")
	}
	if got.WaveDownScore <= got.WaveUpScore {
		t.Fatalf("expected bearish structure, got up=%.4f down=%.4f", got.WaveUpScore, got.WaveDownScore)
	}
}

func TestExtractWaveStructureFeaturesDeepCorrectionCollapsesScore(t *testing.T) {
	bars := []Bar{
		testBar(0, 100, 101, 99, 100, 100),
		testBar(1, 100, 112, 104, 111, 100),
		testBar(2, 111, 106, 96, 103, 100),
		testBar(3, 103, 121, 104, 119, 100),
		testBar(4, 119, 108, 90, 96, 100),
		testBar(5, 96, 126, 97, 124, 100),
		testBar(6, 124, 118, 112, 116, 100),
		testBar(7, 116, 119, 110, 113, 100),
	}

	got := ExtractWaveStructureFeatures(bars, len(bars)-1, 1, len(bars), 0.01)
	if got.WaveUpScore >= 0.6 {
		t.Fatalf("expected deep correction to suppress bullish score, got %.4f", got.WaveUpScore)
	}
}

func TestExtractWaveStructureFeaturesNeedsEnoughPivots(t *testing.T) {
	bars := []Bar{
		testBar(0, 100, 101, 99, 100, 100),
		testBar(1, 100, 105, 99, 104, 100),
		testBar(2, 104, 103, 98, 99, 100),
		testBar(3, 99, 106, 100, 105, 100),
	}

	got := ExtractWaveStructureFeatures(bars, len(bars)-1, 1, len(bars), 0.01)
	if got.ValidStructure {
		t.Fatalf("expected invalid structure with insufficient pivots")
	}
}

func TestExtractLevelClusterFeaturesMergesRepeatedLows(t *testing.T) {
	bars := []Bar{
		testBar(0, 110, 111, 107, 108, 100),
		testBar(1, 108, 109, 99, 101, 100),
		testBar(2, 101, 118, 103, 117, 100),
		testBar(3, 111, 112, 100, 102, 100),
		testBar(4, 102, 119, 104, 118, 100),
		testBar(5, 117, 118, 99.5, 101, 100),
		testBar(6, 101, 120.5, 103, 120, 100),
		testBar(7, 120, 115, 109, 110, 100),
		testBar(8, 110, 124, 112, 123, 100),
	}

	got := ExtractLevelClusterFeatures(bars, len(bars)-1, 1, len(bars), 0.02)
	if got.SupportClusterCount < 1 {
		t.Fatalf("expected support cluster, got %+v", got)
	}
	if got.SupportBounceCount < 2 {
		t.Fatalf("expected merged support bounces, got %+v", got)
	}
	if !got.RecentBreakoutFlag {
		t.Fatalf("expected breakout flag, got %+v", got)
	}
}

func TestExtractLevelClusterFeaturesSplitsWideZones(t *testing.T) {
	bars := []Bar{
		testBar(0, 110, 111, 107, 108, 100),
		testBar(1, 108, 109, 99, 101, 100),
		testBar(2, 101, 118, 103, 117, 100),
		testBar(3, 111, 112, 92, 94, 100),
		testBar(4, 94, 118, 95, 117, 100),
		testBar(5, 117, 118, 98, 100, 100),
		testBar(6, 100, 120, 102, 119, 100),
		testBar(7, 119, 114, 106, 108, 100),
	}

	got := ExtractLevelClusterFeatures(bars, len(bars)-1, 1, len(bars), 0.02)
	if got.SupportClusterCount < 2 {
		t.Fatalf("expected two support clusters, got %+v", got)
	}
}

func TestExtractFibConfluenceFeaturesFindsFiftyPercentPullback(t *testing.T) {
	bars := []Bar{
		testBar(0, 105, 106, 104, 105, 100),
		testBar(1, 105, 106, 98, 100, 100),
		testBar(2, 100, 110, 101, 109, 100),
		testBar(3, 109, 122, 110, 120, 100),
		testBar(4, 120, 112, 108.5, 109.5, 100),
		testBar(5, 109.5, 114, 109, 113, 100),
	}
	levels := LevelClusterFeatures{
		supportLower: 109,
		supportUpper: 110,
	}

	got := ExtractFibConfluenceFeatures(bars, 4, 1, len(bars), 0.02, levels)
	if !got.ValidFibContext {
		t.Fatalf("expected valid fib context")
	}
	if got.Fib500DistancePct >= got.Fib382DistancePct || got.Fib500DistancePct >= got.Fib618DistancePct {
		t.Fatalf("expected 50%% level to be closest, got %+v", got)
	}
	if got.FibClusterOverlapScore <= 0 {
		t.Fatalf("expected fib/cluster overlap, got %+v", got)
	}
}

func TestExtractFibConfluenceFeaturesRejectsTinySwing(t *testing.T) {
	bars := []Bar{
		testBar(0, 100, 100.5, 99.8, 100, 100),
		testBar(1, 100, 101, 99.9, 100.8, 100),
		testBar(2, 100.8, 100.9, 99.7, 100, 100),
		testBar(3, 100, 101.1, 99.8, 100.9, 100),
		testBar(4, 100.9, 101.0, 99.9, 100.2, 100),
	}

	got := ExtractFibConfluenceFeatures(bars, len(bars)-1, 1, len(bars), 0.02, LevelClusterFeatures{})
	if got.ValidFibContext {
		t.Fatalf("expected tiny swing to be ignored, got %+v", got)
	}
}

func TestExtractPriceActionTriggerFeaturesDetectsBullishEngulfing(t *testing.T) {
	bars := []Bar{
		testBar(0, 110, 111, 99, 100, 100),
		testBar(1, 99, 113, 98, 112, 100),
	}

	got := ExtractPriceActionTriggerFeatures(bars, 1, 1, LevelClusterFeatures{}, FibConfluenceFeatures{})
	if !got.BullishEngulfingFlag {
		t.Fatalf("expected bullish engulfing, got %+v", got)
	}
}

func TestExtractPriceActionTriggerFeaturesDetectsBearishEngulfing(t *testing.T) {
	bars := []Bar{
		testBar(0, 100, 111, 99, 110, 100),
		testBar(1, 111, 112, 97, 98, 100),
	}

	got := ExtractPriceActionTriggerFeatures(bars, 1, 1, LevelClusterFeatures{}, FibConfluenceFeatures{})
	if !got.BearishEngulfingFlag {
		t.Fatalf("expected bearish engulfing, got %+v", got)
	}
}

func TestExtractPriceActionTriggerFeaturesScoresBullishPinNearSupport(t *testing.T) {
	bars := []Bar{
		testBar(0, 105, 106, 101, 104, 100),
		testBar(1, 104, 106, 95, 105, 100),
	}
	levels := LevelClusterFeatures{NearestSupportDistancePct: 0.004}

	got := ExtractPriceActionTriggerFeatures(bars, 1, 1, levels, FibConfluenceFeatures{})
	if got.PinBarBullScore <= 0.9 {
		t.Fatalf("expected strong bullish pin score, got %+v", got)
	}
	if got.TriggerQualityScore <= 0 {
		t.Fatalf("expected non-zero trigger quality, got %+v", got)
	}
}

func TestExtractPriceActionTriggerFeaturesPenalizesMidRangeClose(t *testing.T) {
	bars := []Bar{
		testBar(0, 100, 101, 99, 100, 100),
		testBar(1, 100, 105, 95, 100, 100),
	}

	got := ExtractPriceActionTriggerFeatures(bars, 1, 1, LevelClusterFeatures{}, FibConfluenceFeatures{})
	if got.TriggerQualityScore >= 0.3 {
		t.Fatalf("expected weak trigger quality for middle close, got %+v", got)
	}
}

func TestExtractVolumeConfirmationFeaturesDetectsBreakoutSpike(t *testing.T) {
	bars := risingSeries(8, 100, 2, 100)
	bars[len(bars)-1].Volume = 260
	trigger := PriceActionTriggerFeatures{RangeExpansionRatio: 1.3, TriggerQualityScore: 1.4}

	got := ExtractVolumeConfirmationFeatures(bars, len(bars)-1, 5, trigger)
	if !got.VolumeAvailable {
		t.Fatalf("expected volume data to be available")
	}
	if got.VolumeZScore <= 0 {
		t.Fatalf("expected positive z-score, got %+v", got)
	}
	if !got.BreakoutVolumeConfirmed {
		t.Fatalf("expected breakout confirmation, got %+v", got)
	}
}

func TestExtractVolumeConfirmationFeaturesDetectsDryUp(t *testing.T) {
	bars := risingSeries(8, 100, 1, 100)
	bars[len(bars)-1].Close = bars[len(bars)-2].Close - 1
	bars[len(bars)-1].Open = bars[len(bars)-2].Close
	bars[len(bars)-1].High = math.Max(bars[len(bars)-1].Open, bars[len(bars)-1].Close) + 0.5
	bars[len(bars)-1].Low = math.Min(bars[len(bars)-1].Open, bars[len(bars)-1].Close) - 0.5
	bars[len(bars)-1].Volume = 70
	trigger := PriceActionTriggerFeatures{RangeExpansionRatio: 0.8}

	got := ExtractVolumeConfirmationFeatures(bars, len(bars)-1, 5, trigger)
	if !got.PullbackVolumeDryupFlag {
		t.Fatalf("expected dry-up signal, got %+v", got)
	}
}

func TestExtractVolumeConfirmationFeaturesMarksMissingVolume(t *testing.T) {
	bars := risingSeries(6, 100, 1, 0)
	got := ExtractVolumeConfirmationFeatures(bars, len(bars)-1, 5, PriceActionTriggerFeatures{})
	if got.VolumeAvailable {
		t.Fatalf("expected missing volume to stay unavailable, got %+v", got)
	}
}

func TestExtractRegimeTagsTrendUp(t *testing.T) {
	bars := risingSeries(90, 100, 1.5, 100)
	wave := WaveStructureFeatures{WaveUpScore: 1.2, WaveDownScore: 0.2, SwingOverlapRatio: 0.2}

	got := ExtractRegimeTags(bars, len(bars)-1, 15, 60, 14, 90, LevelClusterFeatures{}, wave)
	if !got.TrendUpFlag {
		t.Fatalf("expected uptrend regime, got %+v", got)
	}
}

func TestExtractRegimeTagsTrendDown(t *testing.T) {
	bars := fallingSeries(90, 180, 1.4, 100)
	wave := WaveStructureFeatures{WaveUpScore: 0.2, WaveDownScore: 1.2, SwingOverlapRatio: 0.2}

	got := ExtractRegimeTags(bars, len(bars)-1, 15, 60, 14, 90, LevelClusterFeatures{}, wave)
	if !got.TrendDownFlag {
		t.Fatalf("expected downtrend regime, got %+v", got)
	}
}

func TestExtractRegimeTagsRangeMode(t *testing.T) {
	bars := oscillatingSeries([]float64{100, 100.2, 99.9, 100.1, 99.8, 100, 100.1, 99.9}, 100)
	for len(bars) < 90 {
		base := len(bars)
		for _, close := range []float64{100, 100.15, 99.9, 100.05} {
			open := close
			if len(bars) > 0 {
				open = bars[len(bars)-1].Close
			}
			high := math.Max(open, close) + 0.2
			low := math.Min(open, close) - 0.2
			bars = append(bars, testBar(base, open, high, low, close, 100))
			base++
		}
	}
	wave := WaveStructureFeatures{WaveUpScore: 0.3, WaveDownScore: 0.3, SwingOverlapRatio: 0.85}
	levels := LevelClusterFeatures{SupportClusterCount: 3, ResistanceClusterCount: 3}

	got := ExtractRegimeTags(bars, len(bars)-1, 15, 60, 14, 90, levels, wave)
	if !got.RangeFlag {
		t.Fatalf("expected range regime, got %+v", got)
	}
}

func TestExtractRegimeTagsHighVol(t *testing.T) {
	bars := risingSeries(90, 100, 0.4, 100)
	last := len(bars) - 1
	bars[last].High = bars[last-1].Close + 18
	bars[last].Low = bars[last-1].Close - 18
	bars[last].Close = bars[last-1].Close + 8
	wave := WaveStructureFeatures{WaveUpScore: 1.0, WaveDownScore: 0.1, SwingOverlapRatio: 0.2}

	got := ExtractRegimeTags(bars, last, 15, 60, 14, 90, LevelClusterFeatures{}, wave)
	if !got.HighVolFlag {
		t.Fatalf("expected high-vol regime, got %+v", got)
	}
}

func testBar(day int, open, high, low, close, volume float64) Bar {
	return Bar{
		Time:   time.Date(2024, 1, 1+day, 0, 0, 0, 0, time.UTC),
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: volume,
	}
}

func risingSeries(count int, start, step, volume float64) []Bar {
	bars := make([]Bar, 0, count)
	price := start
	for i := 0; i < count; i++ {
		open := price
		close := price + step
		bars = append(bars, testBar(i, open, close+1, open-1, close, volume))
		price = close
	}
	return bars
}

func fallingSeries(count int, start, step, volume float64) []Bar {
	bars := make([]Bar, 0, count)
	price := start
	for i := 0; i < count; i++ {
		open := price
		close := price - step
		bars = append(bars, testBar(i, open, open+1, close-1, close, volume))
		price = close
	}
	return bars
}

func oscillatingSeries(closes []float64, volume float64) []Bar {
	bars := make([]Bar, 0, len(closes))
	for i, close := range closes {
		open := close
		if i > 0 {
			open = closes[i-1]
		}
		high := math.Max(open, close) + 0.6
		low := math.Min(open, close) - 0.6
		bars = append(bars, testBar(i, open, high, low, close, volume))
	}
	return bars
}
