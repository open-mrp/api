package scheduling

import "time"

// SubtractBusinessDays walks back n business days from a date, skipping weekends.
//
// Carriers count transit in business days: a service quoted at 3 days and handed over on a Thursday delivers the following Tuesday, not Sunday. Subtracting calendar days instead would put every ship-by date up to two days late in exactly the cases that matter, since a week has more weekday-crossing lanes than not.
//
// Holidays are not modelled. There is no holiday calendar in the schema, and a hardcoded one would be wrong for every account that ships internationally or runs through a shutdown week. Weekends alone recover most of the error; the residual is a day around a handful of dates a year, and it errs toward shipping early.
//
// n <= 0 returns the date unchanged, so a lane with no transit means ship-by is the delivery date.
func SubtractBusinessDays(from time.Time, n int) time.Time {
	d := from
	for range n {
		d = d.AddDate(0, 0, -1)
		for isWeekend(d) {
			d = d.AddDate(0, 0, -1)
		}
	}
	return d
}

func isWeekend(t time.Time) bool {
	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return true
	default:
		return false
	}
}
