package promotion

import "time"

func Rollback(request Request, now time.Time, reason string) (Request, error) {
	return advanceRequest(ActionRollback, request, now, reason)
}
