package domain

import "time"

// CreatedRange is a list's created-date filter: After is inclusive, Before exclusive, and either may be nil.
type CreatedRange struct {
	After  *time.Time
	Before *time.Time
}
