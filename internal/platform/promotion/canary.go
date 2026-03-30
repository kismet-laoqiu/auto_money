package promotion

import "time"

func StartCanary(request Request, now time.Time) (Request, error) {
	return advanceRequest(ActionStartCanary, request, now, "")
}

func DegradeCanary(request Request, now time.Time, reason string) (Request, error) {
	return advanceRequest(ActionDegradeCanary, request, now, reason)
}

func Approve(request Request, now time.Time) (Request, error) {
	return advanceRequest(ActionApprove, request, now, "")
}
