package domain

import "time"

// OperatingCalendar is the set of days one party to a shipment operates.
type OperatingCalendar struct {
	ID        string `audit:"id"`
	AccountID string `audit:"account_id"`
	Code      string `audit:"code"`
	Name      string `audit:"name"`
	KindCode  string `audit:"operating_calendar_kind_code"`
	// DaysOfWeek is an ISO Mon-Sun bitmask, '1' for an open day.
	DaysOfWeek string `audit:"days_of_week"`
	// CutoffAt is the local time freight has to be tendered by, as "15:00". Ship calendars only.
	CutoffAt *string `audit:"cutoff_at"`
	// Timezone is the IANA zone the cutoff is read in. Nil on a receive calendar means derive it from the ship-to address.
	Timezone  *string `audit:"timezone"`
	IsDefault bool    `audit:"is_default"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// Closures are the calendar's dated shutdowns, populated only when a caller asked for a window of them.
	Closures []OperatingCalendarClosure
}

// OperatingCalendarClosure is one date a calendar is shut.
type OperatingCalendarClosure struct {
	ID         string    `audit:"id"`
	AccountID  string    `audit:"account_id"`
	CalendarID string    `audit:"operating_calendar_id"`
	ClosedOn   time.Time `audit:"closed_on"`
	Name       string    `audit:"name"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateOperatingCalendarParams is a new calendar.
type CreateOperatingCalendarParams struct {
	ID         string
	AccountID  string
	Code       string
	Name       string
	KindCode   string
	DaysOfWeek string
	CutoffAt   *string
	Timezone   *string
	IsDefault  bool
}

// UpdateOperatingCalendarParams patches a calendar. A nil field is left alone; the Clear flags are how a cutoff or zone is removed, since nil already means "unchanged".
type UpdateOperatingCalendarParams struct {
	ID            string
	AccountID     string
	Name          *string
	DaysOfWeek    *string
	CutoffAt      *string
	ClearCutoffAt bool
	Timezone      *string
	ClearTimezone bool
	IsDefault     *bool
}

// ListOperatingCalendarsParams filters a listing. A nil KindCode returns both kinds.
type ListOperatingCalendarsParams struct {
	AccountID string
	KindCode  *string
}

// ReceiveCalendarQuery is the destination whose receiving days are wanted. Every field but the account is optional: an order with no customer relation and no address still resolves to the account default.
type ReceiveCalendarQuery struct {
	AccountID      string
	BuyerAccountID *string
	AddressID      *string
}

// ClosureWindowQuery is a bounded date range of closures across a set of calendars. Bounded on purpose: resolving one commitment needs the months around its ship-by date, never an account's whole history.
type ClosureWindowQuery struct {
	AccountID   string
	CalendarIDs []string
	From        time.Time
	To          time.Time
}

// OperatingCalendarReferences counts what still points at a calendar, so a delete can explain what it would break rather than silently returning links to a missing row.
type OperatingCalendarReferences struct {
	Addresses int64
	Customers int64
	Groups    int64
	Settings  int64
}

// Total is how many links exist in all.
func (r OperatingCalendarReferences) Total() int64 {
	return r.Addresses + r.Customers + r.Groups + r.Settings
}

// UpsertClosureParams is one closure to write. Re-seeding a year is idempotent and leaves an operator's own label intact.
type UpsertClosureParams struct {
	ID         string
	AccountID  string
	CalendarID string
	ClosedOn   time.Time
	Name       string
}
