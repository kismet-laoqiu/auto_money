package features

import (
	"fmt"
	"strconv"
	"strings"

	"quantlab/internal/core"
)

type Version struct {
	FeatureSet         string `json:"feature_set" yaml:"feature_set"`
	WaveStructure      string `json:"wave_structure" yaml:"wave_structure"`
	LevelCluster       string `json:"level_cluster" yaml:"level_cluster"`
	FibConfluence      string `json:"fib_confluence" yaml:"fib_confluence"`
	PriceActionTrigger string `json:"price_action_trigger" yaml:"price_action_trigger"`
	VolumeConfirmation string `json:"volume_confirmation" yaml:"volume_confirmation"`
	RegimeTags         string `json:"regime_tags" yaml:"regime_tags"`
}

func DefaultVersion() Version {
	return Version{
		FeatureSet:         core.FeatureSetVersion,
		WaveStructure:      "wave-structure.v1",
		LevelCluster:       "level-cluster.v1",
		FibConfluence:      "fib-confluence.v1",
		PriceActionTrigger: "price-action-trigger.v1",
		VolumeConfirmation: "volume-confirmation.v1",
		RegimeTags:         "regime-tags.v1",
	}
}

func (version Version) String() string {
	parts := []string{
		version.FeatureSet,
		version.WaveStructure,
		version.LevelCluster,
		version.FibConfluence,
		version.PriceActionTrigger,
		version.VolumeConfirmation,
		version.RegimeTags,
	}
	return strings.Join(parts, "|")
}

func ParseVersion(value string) (Version, error) {
	parts := strings.Split(strings.TrimSpace(value), "|")
	if len(parts) != 7 {
		return Version{}, fmt.Errorf("invalid feature version %q", value)
	}
	version := Version{
		FeatureSet:         parts[0],
		WaveStructure:      parts[1],
		LevelCluster:       parts[2],
		FibConfluence:      parts[3],
		PriceActionTrigger: parts[4],
		VolumeConfirmation: parts[5],
		RegimeTags:         parts[6],
	}
	for _, item := range []string{version.FeatureSet, version.WaveStructure, version.LevelCluster, version.FibConfluence, version.PriceActionTrigger, version.VolumeConfirmation, version.RegimeTags} {
		if item == "" {
			return Version{}, fmt.Errorf("invalid feature version %q", value)
		}
	}
	return version, nil
}

func CompareVersion(left, right Version) int {
	leftParts := []string{left.FeatureSet, left.WaveStructure, left.LevelCluster, left.FibConfluence, left.PriceActionTrigger, left.VolumeConfirmation, left.RegimeTags}
	rightParts := []string{right.FeatureSet, right.WaveStructure, right.LevelCluster, right.FibConfluence, right.PriceActionTrigger, right.VolumeConfirmation, right.RegimeTags}
	for index := range leftParts {
		cmp := compareVersionToken(leftParts[index], rightParts[index])
		if cmp != 0 {
			return cmp
		}
	}
	return 0
}

func compareVersionToken(left, right string) int {
	if left == right {
		return 0
	}
	leftPrefix, leftNumber := splitVersionToken(left)
	rightPrefix, rightNumber := splitVersionToken(right)
	if leftPrefix != rightPrefix {
		if leftPrefix < rightPrefix {
			return -1
		}
		return 1
	}
	if leftNumber != rightNumber {
		if leftNumber < rightNumber {
			return -1
		}
		return 1
	}
	if left < right {
		return -1
	}
	return 1
}

func splitVersionToken(value string) (string, int) {
	index := strings.LastIndex(value, ".v")
	if index < 0 {
		return value, 0
	}
	number, err := strconv.Atoi(value[index+2:])
	if err != nil {
		return value, 0
	}
	return value[:index], number
}
