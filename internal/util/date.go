package util

import "time"

// BusinessDaysAgo returns the date that is `days` business days before `from`.
// Weekends (Saturday and Sunday) are skipped and the result is normalized to midnight.
func BusinessDaysAgo(from time.Time, days int) time.Time {
	date := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	for days > 0 {
		date = date.AddDate(0, 0, -1)
		if wd := date.Weekday(); wd != time.Saturday && wd != time.Sunday {
			days--
		}
	}
	return date
}
