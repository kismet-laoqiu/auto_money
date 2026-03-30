package promotion

import (
	"fmt"
	"time"
)

func StartShadow(request Request, now time.Time) (Request, error) {
	return advanceRequest(ActionStartShadow, request, now, "")
}

func PassShadow(request Request, now time.Time) (Request, error) {
	return advanceRequest(ActionPassShadow, request, now, "")
}

func transitionRequest(action Action, request Request, now time.Time, reason string) (Request, error) {
	switch action {
	case ActionStartShadow:
		return StartShadow(request, now)
	case ActionPassShadow:
		return PassShadow(request, now)
	case ActionStartCanary:
		return StartCanary(request, now)
	case ActionDegradeCanary:
		return DegradeCanary(request, now, reason)
	case ActionApprove:
		return Approve(request, now)
	case ActionRollback:
		return Rollback(request, now, reason)
	default:
		return Request{}, fmt.Errorf("unsupported action %s", action)
	}
}

func advanceRequest(action Action, request Request, now time.Time, reason string) (Request, error) {
	next, err := Advance(request.State, action)
	if err != nil {
		return Request{}, err
	}
	request.State = next
	request.UpdatedAt = now.UTC()
	request.Title, request.Summary, request.Details = renderTransition(action, request, reason)
	return request, nil
}
