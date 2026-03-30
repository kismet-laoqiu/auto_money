package notifier

import "time"

const DailySummaryTimezone = "Asia/Shanghai"

type DailySummarySchedule struct {
	LocationName string `json:"location_name"`
	Hour         int    `json:"hour"`
	Minute       int    `json:"minute"`
}

func DefaultDailySummarySchedule() DailySummarySchedule {
	return DailySummarySchedule{
		LocationName: DailySummaryTimezone,
		Hour:         20,
		Minute:       0,
	}
}

func (schedule DailySummarySchedule) NextAfter(now time.Time) (time.Time, error) {
	if schedule.LocationName == "" {
		schedule = DefaultDailySummarySchedule()
	}
	location, err := time.LoadLocation(schedule.LocationName)
	if err != nil {
		return time.Time{}, err
	}
	localNow := now.In(location)
	next := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), schedule.Hour, schedule.Minute, 0, 0, location)
	if !next.After(localNow) {
		next = next.AddDate(0, 0, 1)
	}
	return next.UTC(), nil
}
