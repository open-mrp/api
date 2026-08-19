package scheduling

import "time"

// Closure is one dated shutdown on a calendar.
type Closure struct {
	Date time.Time
	Name string
}

// USFederalHolidays returns the eleven federal holidays for a year, on the dates they are observed.
//
// Computed rather than listed because most of them float: Martin Luther King Jr. Day is the third Monday in January, Thanksgiving the fourth Thursday in November, and a hardcoded table would be wrong from the year after it was written. The four fixed-date holidays are shifted to the adjacent weekday when they fall at a weekend, which is what the observance rule does and what carriers and banks actually follow — Independence Day on a Saturday closes the Friday before.
//
// This is a seed, not a rule. It is written into an account's calendars as ordinary editable closures, because a plant that runs through Columbus Day or shuts the whole week of Christmas needs to say so, and nothing about the federal list is binding on a private factory.
func USFederalHolidays(year int) []Closure {
	return []Closure{
		{observed(date(year, time.January, 1)), "New Year's Day"},
		{nthWeekdayOfMonth(year, time.January, time.Monday, 3), "Martin Luther King Jr. Day"},
		{nthWeekdayOfMonth(year, time.February, time.Monday, 3), "Presidents' Day"},
		{lastWeekdayOfMonth(year, time.May, time.Monday), "Memorial Day"},
		{observed(date(year, time.June, 19)), "Juneteenth"},
		{observed(date(year, time.July, 4)), "Independence Day"},
		{nthWeekdayOfMonth(year, time.September, time.Monday, 1), "Labor Day"},
		{nthWeekdayOfMonth(year, time.October, time.Monday, 2), "Columbus Day"},
		{observed(date(year, time.November, 11)), "Veterans Day"},
		{nthWeekdayOfMonth(year, time.November, time.Thursday, 4), "Thanksgiving Day"},
		{observed(date(year, time.December, 25)), "Christmas Day"},
	}
}

// observed moves a fixed-date holiday off a weekend: Saturday is kept on the Friday before, Sunday on the Monday after.
func observed(d time.Time) time.Time {
	switch d.Weekday() {
	case time.Saturday:
		return d.AddDate(0, 0, -1)
	case time.Sunday:
		return d.AddDate(0, 0, 1)
	default:
		return d
	}
}

// nthWeekdayOfMonth is the nth occurrence of a weekday in a month, counting from one.
func nthWeekdayOfMonth(year int, month time.Month, weekday time.Weekday, n int) time.Time {
	first := date(year, month, 1)
	offset := (int(weekday) - int(first.Weekday()) + 7) % 7
	return first.AddDate(0, 0, offset+(n-1)*7)
}

// lastWeekdayOfMonth is the final occurrence of a weekday in a month.
func lastWeekdayOfMonth(year int, month time.Month, weekday time.Weekday) time.Time {
	// The zeroth day of the following month is the last day of this one, which avoids special-casing month lengths and leap years.
	last := date(year, month+1, 0)
	offset := (int(last.Weekday()) - int(weekday) + 7) % 7
	return last.AddDate(0, 0, -offset)
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
