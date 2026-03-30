package promotion

import "testing"

func TestAdvanceHappyPath(t *testing.T) {
	state := StateBacktestPassed
	var err error

	state, err = Advance(state, ActionStartShadow)
	if err != nil {
		t.Fatalf("start shadow: %v", err)
	}
	if state != StateShadowRunning {
		t.Fatalf("unexpected shadow running state: %s", state)
	}

	state, err = Advance(state, ActionPassShadow)
	if err != nil {
		t.Fatalf("pass shadow: %v", err)
	}
	if state != StateShadowPassed {
		t.Fatalf("unexpected shadow passed state: %s", state)
	}

	state, err = Advance(state, ActionStartCanary)
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	if state != StateCanaryRunning {
		t.Fatalf("unexpected canary running state: %s", state)
	}

	state, err = Advance(state, ActionApprove)
	if err != nil {
		t.Fatalf("approve live: %v", err)
	}
	if state != StateLiveActive {
		t.Fatalf("unexpected live state: %s", state)
	}
}

func TestAdvanceRejectsSkippedGates(t *testing.T) {
	if _, err := Advance(StateBacktestPassed, ActionStartCanary); err == nil {
		t.Fatalf("expected start canary to reject skipped shadow")
	}
	if _, err := Advance(StateShadowRunning, ActionApprove); err == nil {
		t.Fatalf("expected approve to reject running shadow")
	}
	if _, err := Advance(StateCanaryDegraded, ActionApprove); err == nil {
		t.Fatalf("expected approve to reject degraded canary")
	}
}
