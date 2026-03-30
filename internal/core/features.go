package core

import (
	"math"

	"quantlab/internal/config"
)

const FeatureSetVersion = "feature-set.v1"

type WaveStructureFeatures struct {
	WaveUpScore           float64 `json:"wave_up_score"`
	WaveDownScore         float64 `json:"wave_down_score"`
	ImpulseExtensionRatio float64 `json:"impulse_extension_ratio"`
	CorrectiveDepthRatio  float64 `json:"corrective_depth_ratio"`
	SwingOverlapRatio     float64 `json:"swing_overlap_ratio"`
	StructureAgeBars      int     `json:"structure_age_bars"`
	ValidStructure        bool    `json:"valid_structure"`
}

type LevelClusterFeatures struct {
	NearestSupportDistancePct    float64 `json:"nearest_support_distance_pct"`
	NearestResistanceDistancePct float64 `json:"nearest_resistance_distance_pct"`
	SupportClusterCount          int     `json:"support_cluster_count"`
	ResistanceClusterCount       int     `json:"resistance_cluster_count"`
	SupportBounceCount           int     `json:"support_bounce_count"`
	ResistanceBounceCount        int     `json:"resistance_bounce_count"`
	ZoneWidthPct                 float64 `json:"zone_width_pct"`
	RecentBreakoutFlag           bool    `json:"recent_breakout_flag"`

	supportLower    float64
	supportUpper    float64
	resistanceLower float64
	resistanceUpper float64
}

type FibConfluenceFeatures struct {
	Fib382DistancePct      float64 `json:"fib_382_distance_pct"`
	Fib500DistancePct      float64 `json:"fib_500_distance_pct"`
	Fib618DistancePct      float64 `json:"fib_618_distance_pct"`
	FibZoneHitCount        int     `json:"fib_zone_hit_count"`
	FibClusterOverlapScore float64 `json:"fib_cluster_overlap_score"`
	ActiveSwingDirection   string  `json:"active_swing_direction"`
	ValidFibContext        bool    `json:"valid_fib_context"`
}

type PriceActionTriggerFeatures struct {
	BullishEngulfingFlag bool    `json:"bullish_engulfing_flag"`
	BearishEngulfingFlag bool    `json:"bearish_engulfing_flag"`
	PinBarBullScore      float64 `json:"pin_bar_bull_score"`
	PinBarBearScore      float64 `json:"pin_bar_bear_score"`
	CloseLocationValue   float64 `json:"close_location_value"`
	RangeExpansionRatio  float64 `json:"range_expansion_ratio"`
	BreakRetestFlag      bool    `json:"break_retest_flag"`
	TriggerQualityScore  float64 `json:"trigger_quality_score"`
	RSI14                float64 `json:"rsi14"`
	NeedleDropPct        float64 `json:"needle_drop_pct"`
	ReclaimPct           float64 `json:"reclaim_pct"`
}

type VolumeConfirmationFeatures struct {
	VolumeZScore            float64 `json:"volume_zscore"`
	RelativeVolumeRatio     float64 `json:"relative_volume_ratio"`
	PullbackVolumeDryupFlag bool    `json:"pullback_volume_dryup_flag"`
	BreakoutVolumeConfirmed bool    `json:"breakout_volume_confirmed"`
	VolumeAvailable         bool    `json:"volume_available"`
}

type RegimeTags struct {
	TrendUpFlag     bool    `json:"trend_up_flag"`
	TrendDownFlag   bool    `json:"trend_down_flag"`
	RangeFlag       bool    `json:"range_flag"`
	HighVolFlag     bool    `json:"high_vol_flag"`
	CompressionFlag bool    `json:"compression_flag"`
	RegimeScore     float64 `json:"regime_score"`
}

type FeatureSet struct {
	Wave    WaveStructureFeatures      `json:"wave"`
	Levels  LevelClusterFeatures       `json:"levels"`
	Fib     FibConfluenceFeatures      `json:"fib"`
	Trigger PriceActionTriggerFeatures `json:"trigger"`
	Volume  VolumeConfirmationFeatures `json:"volume"`
	Regime  RegimeTags                 `json:"regime"`
}

type directionalWaveMetrics struct {
	Score       float64
	Extension   float64
	Depth       float64
	Overlap     float64
	OldestIndex int
}

type swingSegment struct {
	Start     Pivot
	End       Pivot
	Direction string
	Range     float64
}

type levelZone struct {
	Lower           float64
	Upper           float64
	Mid             float64
	SupportCount    int
	ResistanceCount int
	LastIndex       int
}

func ExtractFeatureSet(bars []Bar, idx int, cfg config.StrategyConfig) FeatureSet {
	lookback := maxInt(cfg.LevelLookback, maxInt(cfg.SlowSMA*2, cfg.ATRWindow*4))
	minSwingPct := math.Max(cfg.LevelTolerance*1.5, 0.02)
	levels := ExtractLevelClusterFeatures(bars, idx, cfg.PivotWindow, cfg.LevelLookback, cfg.LevelTolerance)
	wave := ExtractWaveStructureFeatures(bars, idx, cfg.PivotWindow, lookback, minSwingPct)
	fib := ExtractFibConfluenceFeatures(bars, idx, cfg.PivotWindow, lookback, cfg.FibTolerance, levels)
	trigger := ExtractPriceActionTriggerFeatures(bars, idx, cfg.ATRWindow, levels, fib)
	volume := ExtractVolumeConfirmationFeatures(bars, idx, maxInt(cfg.ATRWindow*2, 20), trigger)
	regime := ExtractRegimeTags(bars, idx, cfg.FastSMA, cfg.SlowSMA, cfg.ATRWindow, lookback, levels, wave)
	return FeatureSet{
		Wave:    wave,
		Levels:  levels,
		Fib:     fib,
		Trigger: trigger,
		Volume:  volume,
		Regime:  regime,
	}
}

func ExtractWaveStructureFeatures(bars []Bar, idx, pivotWindow, lookback int, minSwingPct float64) WaveStructureFeatures {
	pivots := pivotsInWindow(bars, idx, pivotWindow, lookback, minSwingPct)
	if len(pivots) < 5 {
		return WaveStructureFeatures{}
	}

	metrics := WaveStructureFeatures{ValidStructure: true}
	upMetrics, upOK := scoreDirectionalStructure(pivots, idx, "up")
	downMetrics, downOK := scoreDirectionalStructure(pivots, idx, "down")

	if upOK {
		metrics.WaveUpScore = upMetrics.Score
	}
	if downOK {
		metrics.WaveDownScore = downMetrics.Score
	}

	if !upOK && !downOK {
		upBias, downBias := waveBias(pivots)
		if upBias {
			metrics.WaveUpScore = 0.4
		}
		if downBias {
			metrics.WaveDownScore = 0.4
		}
		if metrics.WaveUpScore == 0 && metrics.WaveDownScore == 0 {
			metrics.ValidStructure = false
			return metrics
		}
		metrics.StructureAgeBars = idx - pivots[len(pivots)-5].Index
		return metrics
	}

	dominant := upMetrics
	if downMetrics.Score > dominant.Score {
		dominant = downMetrics
	}
	metrics.ImpulseExtensionRatio = dominant.Extension
	metrics.CorrectiveDepthRatio = dominant.Depth
	metrics.SwingOverlapRatio = dominant.Overlap
	metrics.StructureAgeBars = idx - dominant.OldestIndex
	return metrics
}

func ExtractLevelClusterFeatures(bars []Bar, idx, pivotWindow, clusterLookback int, levelTolerance float64) LevelClusterFeatures {
	features := LevelClusterFeatures{
		NearestSupportDistancePct:    1,
		NearestResistanceDistancePct: 1,
	}
	pivots := pivotsInWindow(bars, idx, pivotWindow, clusterLookback, math.Max(levelTolerance*0.5, 0.005))
	if len(pivots) == 0 || idx <= 0 {
		return features
	}

	zones := buildLevelZones(pivots, levelTolerance)
	if len(zones) == 0 {
		return features
	}

	price := bars[idx].Close
	prevClose := bars[idx-1].Close
	nearestSupportIndex := -1
	nearestResistanceIndex := -1
	breakoutResistanceIndex := -1
	breakdownSupportIndex := -1
	for i, zone := range zones {
		if zone.SupportCount > 0 {
			features.SupportClusterCount++
			if zone.Mid <= price*(1+levelTolerance) && (nearestSupportIndex == -1 || zone.Mid > zones[nearestSupportIndex].Mid) {
				nearestSupportIndex = i
			}
			if zone.Mid >= price && (breakdownSupportIndex == -1 || zone.Mid < zones[breakdownSupportIndex].Mid) {
				breakdownSupportIndex = i
			}
		}
		if zone.ResistanceCount > 0 {
			features.ResistanceClusterCount++
			if zone.Mid >= price*(1-levelTolerance) && (nearestResistanceIndex == -1 || zone.Mid < zones[nearestResistanceIndex].Mid) {
				nearestResistanceIndex = i
			}
			if zone.Mid <= price && (breakoutResistanceIndex == -1 || zone.Mid > zones[breakoutResistanceIndex].Mid) {
				breakoutResistanceIndex = i
			}
		}
	}

	if nearestSupportIndex >= 0 {
		zone := zones[nearestSupportIndex]
		features.NearestSupportDistancePct = distancePct(price, zone.Mid)
		features.SupportBounceCount = maxInt(0, zone.SupportCount-1)
		features.supportLower = zone.Lower
		features.supportUpper = zone.Upper
	}
	if nearestResistanceIndex >= 0 {
		zone := zones[nearestResistanceIndex]
		features.NearestResistanceDistancePct = distancePct(price, zone.Mid)
		features.ResistanceBounceCount = maxInt(0, zone.ResistanceCount-1)
		features.resistanceLower = zone.Lower
		features.resistanceUpper = zone.Upper
	}

	nearestWidth := 0.0
	if nearestSupportIndex >= 0 {
		zone := zones[nearestSupportIndex]
		nearestWidth = (zone.Upper - zone.Lower) / math.Max(price, 1e-9)
	}
	if nearestResistanceIndex >= 0 {
		zone := zones[nearestResistanceIndex]
		width := (zone.Upper - zone.Lower) / math.Max(price, 1e-9)
		if nearestWidth == 0 || width < nearestWidth {
			nearestWidth = width
		}
	}
	features.ZoneWidthPct = nearestWidth

	if breakoutResistanceIndex >= 0 {
		zone := zones[breakoutResistanceIndex]
		margin := math.Max(zone.Upper-zone.Lower, price*levelTolerance*0.25)
		if zone.ResistanceCount >= 2 && prevClose <= zone.Upper && price > zone.Upper+margin*0.2 {
			features.RecentBreakoutFlag = true
		}
	}
	if !features.RecentBreakoutFlag && breakdownSupportIndex >= 0 {
		zone := zones[breakdownSupportIndex]
		margin := math.Max(zone.Upper-zone.Lower, price*levelTolerance*0.25)
		if zone.SupportCount >= 2 && prevClose >= zone.Lower && price < zone.Lower-margin*0.2 {
			features.RecentBreakoutFlag = true
		}
	}

	return features
}

func ExtractFibConfluenceFeatures(bars []Bar, idx, pivotWindow, swingLookback int, fibTolerance float64, levels LevelClusterFeatures) FibConfluenceFeatures {
	features := FibConfluenceFeatures{
		Fib382DistancePct: 1,
		Fib500DistancePct: 1,
		Fib618DistancePct: 1,
	}
	pivots := pivotsInWindow(bars, idx, pivotWindow, swingLookback, math.Max(fibTolerance, 0.01))
	swings := buildSwingSegments(pivots)
	if len(swings) == 0 {
		return features
	}

	minAmplitude := math.Max(fibTolerance*2.5, 0.02)
	chosen, ok := chooseDominantSwing(swings, idx, swingLookback, minAmplitude)
	if !ok {
		return features
	}

	high := math.Max(chosen.Start.Price, chosen.End.Price)
	low := math.Min(chosen.Start.Price, chosen.End.Price)
	levelsRaw := retracementLevels(high, low, chosen.Direction == "up")
	if len(levelsRaw) != 3 {
		return features
	}

	price := bars[idx].Close
	features.Fib382DistancePct = distancePct(price, levelsRaw[0])
	features.Fib500DistancePct = distancePct(price, levelsRaw[1])
	features.Fib618DistancePct = distancePct(price, levelsRaw[2])
	features.ActiveSwingDirection = chosen.Direction
	features.ValidFibContext = true

	for _, level := range levelsRaw {
		if distancePct(price, level) <= fibTolerance {
			features.FibZoneHitCount++
		}
		features.FibClusterOverlapScore += bandOverlapScore(level, levels.supportLower, levels.supportUpper, fibTolerance)
		features.FibClusterOverlapScore += bandOverlapScore(level, levels.resistanceLower, levels.resistanceUpper, fibTolerance)
	}

	return features
}

func ExtractPriceActionTriggerFeatures(bars []Bar, idx, atrWindow int, levels LevelClusterFeatures, fib FibConfluenceFeatures) PriceActionTriggerFeatures {
	features := PriceActionTriggerFeatures{CloseLocationValue: 0.5}
	if idx <= 0 || idx >= len(bars) {
		return features
	}

	prev := bars[idx-1]
	curr := bars[idx]
	barRange := curr.High - curr.Low
	if barRange > 0 {
		features.CloseLocationValue = (curr.Close - curr.Low) / barRange
	}

	atr := atrAt(bars, idx, atrWindow)
	if atr > 0 {
		features.RangeExpansionRatio = barRange / atr
	}

	features.BullishEngulfingFlag = bullishEngulfing(prev, curr)
	features.BearishEngulfingFlag = bearishEngulfing(prev, curr)
	features.PinBarBullScore = pinBarScore(curr, true)
	features.PinBarBearScore = pinBarScore(curr, false)
	features.RSI14 = rsiAt(bars, idx, 14)
	features.NeedleDropPct = needleDropPct(prev, curr)
	features.ReclaimPct = reclaimRatio(prev, curr)

	if levels.resistanceUpper > 0 {
		margin := math.Max(levels.resistanceUpper-levels.resistanceLower, curr.Close*0.002)
		if prev.Close > levels.resistanceUpper && curr.Low <= levels.resistanceUpper+margin && curr.Close > levels.resistanceUpper {
			features.BreakRetestFlag = true
		}
	}
	if !features.BreakRetestFlag && levels.supportLower > 0 {
		margin := math.Max(levels.supportUpper-levels.supportLower, curr.Close*0.002)
		if prev.Close < levels.supportLower && curr.High >= levels.supportLower-margin && curr.Close < levels.supportLower {
			features.BreakRetestFlag = true
		}
	}

	quality := 0.0
	if features.BullishEngulfingFlag || features.BearishEngulfingFlag {
		quality += 0.85
	}
	quality += math.Max(features.PinBarBullScore, features.PinBarBearScore) * 0.55
	if features.RangeExpansionRatio > 0.8 {
		quality += math.Min(0.7, (features.RangeExpansionRatio-0.8)*0.4)
	}
	if levels.SupportClusterCount > 0 && levels.NearestSupportDistancePct <= 0.02 {
		quality += 0.3
	}
	if levels.ResistanceClusterCount > 0 && levels.NearestResistanceDistancePct <= 0.02 {
		quality += 0.3
	}
	if fib.ValidFibContext && fib.FibZoneHitCount > 0 {
		quality += 0.3 + math.Min(0.4, fib.FibClusterOverlapScore*0.15)
	}
	if features.RSI14 > 0 && features.RSI14 < 28 {
		quality += 0.25
	}
	if features.NeedleDropPct >= 1.5 {
		quality += math.Min(0.35, features.NeedleDropPct*0.08)
	}
	if features.ReclaimPct >= 0.7 {
		quality += 0.25
	}
	if features.CloseLocationValue > 0.35 && features.CloseLocationValue < 0.65 {
		quality -= 0.3
	}
	features.TriggerQualityScore = clamp(quality, 0, 3)
	return features
}

func ExtractVolumeConfirmationFeatures(bars []Bar, idx, volumeWindow int, trigger PriceActionTriggerFeatures) VolumeConfirmationFeatures {
	if idx <= 0 || idx >= len(bars) || volumeWindow <= 0 {
		return VolumeConfirmationFeatures{}
	}

	start := idx - volumeWindow
	if start < 0 {
		start = 0
	}
	history := make([]float64, 0, idx-start)
	for i := start; i < idx; i++ {
		if bars[i].Volume > 0 {
			history = append(history, bars[i].Volume)
		}
	}
	if bars[idx].Volume <= 0 || len(history) < maxInt(3, volumeWindow/3) {
		return VolumeConfirmationFeatures{}
	}

	mean := average(history)
	std := stddev(history, mean)
	relative := 0.0
	if mean > 0 {
		relative = bars[idx].Volume / mean
	}
	zscore := 0.0
	if std > 0 {
		zscore = (bars[idx].Volume - mean) / std
	} else if relative > 0 {
		zscore = relative - 1
	}

	prevClose := bars[idx-1].Close
	movePct := 0.0
	if prevClose > 0 {
		movePct = math.Abs(bars[idx].Close-prevClose) / prevClose
	}

	return VolumeConfirmationFeatures{
		VolumeZScore:            zscore,
		RelativeVolumeRatio:     relative,
		PullbackVolumeDryupFlag: relative > 0 && relative < 0.85 && movePct < 0.03 && trigger.RangeExpansionRatio < 1.0,
		BreakoutVolumeConfirmed: relative >= 1.25 && trigger.RangeExpansionRatio >= 1.0 && (trigger.BreakRetestFlag || trigger.TriggerQualityScore >= 1.2),
		VolumeAvailable:         true,
	}
}

func ExtractRegimeTags(bars []Bar, idx, fastWindow, slowWindow, atrWindow, lookback int, levels LevelClusterFeatures, wave WaveStructureFeatures) RegimeTags {
	if idx <= 1 {
		return RegimeTags{}
	}

	effectiveFastWindow := minInt(fastWindow, idx+1)
	if effectiveFastWindow < 3 {
		return RegimeTags{}
	}
	effectiveSlowWindow := minInt(slowWindow, idx+1)
	if effectiveSlowWindow < effectiveFastWindow {
		effectiveSlowWindow = effectiveFastWindow
	}
	effectiveATRWindow := minInt(atrWindow, idx)
	if effectiveATRWindow < 2 {
		return RegimeTags{}
	}

	fastNow, okFast := smaAt(bars, idx, effectiveFastWindow)
	slowNow, okSlow := smaAt(bars, idx, effectiveSlowWindow)
	if !okFast || !okSlow {
		return RegimeTags{}
	}

	slopeOffset := maxInt(1, minInt(maxInt(lookback/3, 1), effectiveFastWindow))
	pastIdx := idx - slopeOffset
	if pastIdx < effectiveSlowWindow-1 {
		pastIdx = effectiveSlowWindow - 1
	}
	fastPast, okFastPast := smaAt(bars, pastIdx, effectiveFastWindow)
	slowPast, okSlowPast := smaAt(bars, pastIdx, effectiveSlowWindow)
	if !okFastPast || !okSlowPast {
		return RegimeTags{}
	}

	price := bars[idx].Close
	if price <= 0 {
		return RegimeTags{}
	}

	separation := (fastNow - slowNow) / price
	fastSlope := (fastNow - fastPast) / price
	slowSlope := (slowNow - slowPast) / price
	currentATR := atrAt(bars, idx, effectiveATRWindow)
	atrValues := make([]float64, 0, lookback)
	start := idx - lookback + 1
	if start < effectiveATRWindow {
		start = effectiveATRWindow
	}
	for i := start; i <= idx; i++ {
		value := atrAt(bars, i, effectiveATRWindow)
		if value > 0 {
			atrValues = append(atrValues, value)
		}
	}
	atrPercentile := percentileRank(atrValues, currentATR)
	clusterDensity := float64(levels.SupportClusterCount+levels.ResistanceClusterCount) / math.Max(float64(maxInt(lookback/10, 1)), 1)

	enoughTrendHistory := idx+1 >= maxInt(effectiveSlowWindow, 20)
	bullishStructure := wave.ValidStructure && wave.WaveUpScore > wave.WaveDownScore && wave.WaveUpScore > 0.8
	bearishStructure := wave.ValidStructure && wave.WaveDownScore > wave.WaveUpScore && wave.WaveDownScore > 0.8
	trendUp := enoughTrendHistory && separation > 0.004 && fastSlope > 0 && slowSlope >= -0.001 && !bearishStructure
	trendDown := enoughTrendHistory && separation < -0.004 && fastSlope < 0 && slowSlope <= 0.001 && !bullishStructure
	compression := atrPercentile > 0 && atrPercentile < 0.35 && math.Abs(separation) < 0.003
	rangeFlag := !trendUp && !trendDown && (wave.SwingOverlapRatio >= 0.85 || clusterDensity >= 0.25) && atrPercentile < 0.7
	highVol := atrPercentile > 0.65 || currentATR/math.Max(price, 1e-9) > 0.05 || (idx+1 < slowWindow && currentATR/math.Max(price, 1e-9) > 0.03)

	score := 0.3 + math.Min(0.4, atrPercentile*0.3)
	switch {
	case trendUp || trendDown:
		score = 0.8 + math.Min(0.8, math.Abs(separation)*120) + math.Min(0.6, math.Max(wave.WaveUpScore, wave.WaveDownScore)*0.25)
	case rangeFlag:
		score = 0.6 + math.Min(0.5, wave.SwingOverlapRatio*0.4) + math.Min(0.4, clusterDensity)
	}
	if highVol {
		score += 0.2
	}
	if compression {
		score += 0.1
	}

	return RegimeTags{
		TrendUpFlag:     trendUp,
		TrendDownFlag:   trendDown,
		RangeFlag:       rangeFlag,
		HighVolFlag:     highVol,
		CompressionFlag: compression,
		RegimeScore:     clamp(score, 0, 3),
	}
}

func pivotsInWindow(bars []Bar, idx, pivotWindow, lookback int, minSwingPct float64) []Pivot {
	if len(bars) == 0 || idx < 0 || idx >= len(bars) || pivotWindow <= 0 {
		return nil
	}
	start := idx - lookback + 1
	if start < 0 {
		start = 0
	}
	raw := findPivots(bars[start:idx+1], pivotWindow)
	if len(raw) == 0 {
		return nil
	}
	pivots := make([]Pivot, 0, len(raw))
	for _, pivot := range raw {
		pivot.Index += start
		pivots = append(pivots, pivot)
	}
	return normalizePivots(pivots, minSwingPct)
}

func normalizePivots(raw []Pivot, minSwingPct float64) []Pivot {
	if len(raw) == 0 {
		return nil
	}
	pivots := make([]Pivot, 0, len(raw))
	for _, pivot := range raw {
		if len(pivots) == 0 {
			pivots = append(pivots, pivot)
			continue
		}
		last := &pivots[len(pivots)-1]
		if pivot.Kind == last.Kind {
			if pivotMoreExtreme(pivot, *last) {
				*last = pivot
			}
			continue
		}
		if minSwingPct > 0 {
			base := math.Max(math.Abs(last.Price), 1e-9)
			if math.Abs(pivot.Price-last.Price)/base < minSwingPct {
				continue
			}
		}
		pivots = append(pivots, pivot)
	}
	return pivots
}

func pivotMoreExtreme(next, current Pivot) bool {
	if next.Kind == "high" {
		return next.Price >= current.Price
	}
	return next.Price <= current.Price
}

func buildSwingSegments(pivots []Pivot) []swingSegment {
	if len(pivots) < 2 {
		return nil
	}
	swings := make([]swingSegment, 0, len(pivots)-1)
	for i := 1; i < len(pivots); i++ {
		start := pivots[i-1]
		end := pivots[i]
		if start.Kind == end.Kind {
			continue
		}
		direction := "up"
		if end.Price < start.Price {
			direction = "down"
		}
		swings = append(swings, swingSegment{
			Start:     start,
			End:       end,
			Direction: direction,
			Range:     math.Abs(end.Price - start.Price),
		})
	}
	return swings
}

func scoreDirectionalStructure(pivots []Pivot, idx int, direction string) (directionalWaveMetrics, bool) {
	swings := buildSwingSegments(pivots)
	if len(swings) < 3 {
		return directionalWaveMetrics{}, false
	}
	for i := len(swings) - 1; i >= 2; i-- {
		latest := swings[i]
		correction := swings[i-1]
		prior := swings[i-2]
		if latest.Direction != direction || correction.Direction == direction || prior.Direction != direction {
			continue
		}
		if prior.Range <= 0 {
			continue
		}
		score := 0.3
		extension := latest.Range / prior.Range
		depth := correction.Range / prior.Range
		overlap := rangeOverlapRatio(prior.Start.Price, prior.End.Price, correction.Start.Price, correction.End.Price)

		if direction == "up" {
			if latest.End.Price > prior.End.Price {
				score += 0.45
			} else {
				score -= 0.25
			}
			if correction.End.Price > prior.Start.Price {
				score += 0.35
			} else {
				score -= 0.85
			}
		} else {
			if latest.End.Price < prior.End.Price {
				score += 0.45
			} else {
				score -= 0.25
			}
			if correction.End.Price < prior.Start.Price {
				score += 0.35
			} else {
				score -= 0.85
			}
		}

		if extension >= 0.75 {
			score += math.Min(0.7, (extension-0.75)*0.8)
		} else {
			score -= 0.15
		}
		switch {
		case depth <= 0.75:
			score += 0.45
		case depth <= 1.0:
			score += 0.15
		default:
			score -= math.Min(0.9, (depth-1.0)*1.2)
		}
		score -= overlap * 0.25
		if score < 0 {
			score = 0
		}
		return directionalWaveMetrics{
			Score:       score,
			Extension:   extension,
			Depth:       depth,
			Overlap:     overlap,
			OldestIndex: prior.Start.Index,
		}, true
	}
	return directionalWaveMetrics{}, false
}

func rangeOverlapRatio(aStart, aEnd, bStart, bEnd float64) float64 {
	aLow := math.Min(aStart, aEnd)
	aHigh := math.Max(aStart, aEnd)
	bLow := math.Min(bStart, bEnd)
	bHigh := math.Max(bStart, bEnd)
	lower := math.Max(aLow, bLow)
	upper := math.Min(aHigh, bHigh)
	if upper <= lower {
		return 0
	}
	base := math.Min(aHigh-aLow, bHigh-bLow)
	if base <= 0 {
		return 0
	}
	return (upper - lower) / base
}

func buildLevelZones(pivots []Pivot, levelTolerance float64) []levelZone {
	zones := make([]levelZone, 0, len(pivots))
	for _, pivot := range pivots {
		match := -1
		bestDistance := levelTolerance
		for i, zone := range zones {
			distance := distancePct(pivot.Price, zone.Mid)
			if distance <= levelTolerance && distance <= bestDistance {
				match = i
				bestDistance = distance
			}
		}
		if match == -1 {
			zone := levelZone{Lower: pivot.Price, Upper: pivot.Price, Mid: pivot.Price, LastIndex: pivot.Index}
			if pivot.Kind == "low" {
				zone.SupportCount = 1
			} else {
				zone.ResistanceCount = 1
			}
			zones = append(zones, zone)
			continue
		}
		zone := &zones[match]
		if pivot.Price < zone.Lower {
			zone.Lower = pivot.Price
		}
		if pivot.Price > zone.Upper {
			zone.Upper = pivot.Price
		}
		total := zone.SupportCount + zone.ResistanceCount
		zone.Mid = (zone.Mid*float64(total) + pivot.Price) / float64(total+1)
		if pivot.Kind == "low" {
			zone.SupportCount++
		} else {
			zone.ResistanceCount++
		}
		zone.LastIndex = pivot.Index
	}
	return zones
}

func chooseDominantSwing(swings []swingSegment, idx, swingLookback int, minAmplitude float64) (swingSegment, bool) {
	for i := len(swings) - 1; i >= 0; i-- {
		swing := swings[i]
		base := math.Max(math.Min(math.Abs(swing.Start.Price), math.Abs(swing.End.Price)), 1e-9)
		if swing.Range/base < minAmplitude {
			continue
		}
		if idx-swing.End.Index > swingLookback {
			continue
		}
		return swing, true
	}
	return swingSegment{}, false
}

func bandOverlapScore(level, lower, upper, tolerance float64) float64 {
	if lower == 0 && upper == 0 {
		return 0
	}
	bandLower := level * (1 - tolerance)
	bandUpper := level * (1 + tolerance)
	overlapLower := math.Max(bandLower, lower)
	overlapUpper := math.Min(bandUpper, upper)
	if overlapUpper <= overlapLower {
		return 0
	}
	base := bandUpper - bandLower
	if base <= 0 {
		return 0
	}
	return (overlapUpper - overlapLower) / base
}

func pinBarScore(bar Bar, bullish bool) float64 {
	total := bar.High - bar.Low
	if total <= 0 {
		return 0
	}
	body := math.Abs(bar.Close - bar.Open)
	upper := bar.High - math.Max(bar.Close, bar.Open)
	lower := math.Min(bar.Close, bar.Open) - bar.Low
	if bullish {
		score := lower/total - body/total*0.25 - upper/total*0.8
		if bullishPin(bar) {
			score = math.Max(score, 1.0)
		}
		return clamp(score, 0, 1.5)
	}
	score := upper/total - body/total*0.25 - lower/total*0.8
	if bearishPin(bar) {
		score = math.Max(score, 1.0)
	}
	return clamp(score, 0, 1.5)
}

func rsiAt(bars []Bar, idx, window int) float64 {
	if idx <= 0 || window <= 0 || idx < window {
		return 0
	}
	gains := 0.0
	losses := 0.0
	for i := idx - window + 1; i <= idx; i++ {
		change := bars[i].Close - bars[i-1].Close
		if change > 0 {
			gains += change
			continue
		}
		losses -= change
	}
	if losses == 0 {
		if gains == 0 {
			return 50
		}
		return 100
	}
	rs := gains / losses
	return 100 - 100/(1+rs)
}

func needleDropPct(prev, curr Bar) float64 {
	ref := curr.Open
	if prev.Close > ref {
		ref = prev.Close
	}
	if ref <= 0 || curr.Low >= ref {
		return 0
	}
	return (ref - curr.Low) / ref * 100
}

func reclaimRatio(prev, curr Bar) float64 {
	ref := curr.Open
	if prev.Close > ref {
		ref = prev.Close
	}
	drop := ref - curr.Low
	if drop <= 0 {
		return 0
	}
	return clamp((curr.Close-curr.Low)/drop, 0, 1)
}

func percentileRank(values []float64, current float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minValue := values[0]
	maxValue := values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	if maxValue == minValue {
		return 0.5
	}
	count := 0
	for _, value := range values {
		if value <= current {
			count++
		}
	}
	return float64(count) / float64(len(values))
}

func distancePct(price, level float64) float64 {
	if price == 0 || level == 0 {
		return 1
	}
	return math.Abs(price-level) / math.Abs(price)
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
