package features

import "testing"

func TestDefaultVersionRoundTripAndCompare(t *testing.T) {
	want := Version{
		FeatureSet:         "feature-set.v1",
		WaveStructure:      "wave-structure.v1",
		LevelCluster:       "level-cluster.v1",
		FibConfluence:      "fib-confluence.v1",
		PriceActionTrigger: "price-action-trigger.v1",
		VolumeConfirmation: "volume-confirmation.v1",
		RegimeTags:         "regime-tags.v1",
	}
	got := DefaultVersion()
	if got != want {
		t.Fatalf("unexpected default version: %+v", got)
	}
	serialized := got.String()
	parsed, err := ParseVersion(serialized)
	if err != nil {
		t.Fatalf("parse version %q: %v", serialized, err)
	}
	if parsed != want {
		t.Fatalf("unexpected parsed version: %+v", parsed)
	}
	next := got
	next.PriceActionTrigger = "price-action-trigger.v2"
	if CompareVersion(next, got) <= 0 {
		t.Fatalf("expected %q to compare newer than %q", next.String(), got.String())
	}
	if CompareVersion(got, next) >= 0 {
		t.Fatalf("expected %q to compare older than %q", got.String(), next.String())
	}
	if CompareVersion(got, got) != 0 {
		t.Fatalf("expected identical versions to compare equal")
	}
}
