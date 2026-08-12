package scheduling

import (
	"time"

	"github.com/augno/api/shared/constants"
)

// Lead-time sources, aliased from the shared enum so the engine cannot drift from the API contract.
const (
	LeadTimeSourceCustomer     = string(constants.LeadTimeSourceCustomer)
	LeadTimeSourceAccountGroup = string(constants.LeadTimeSourceAccountGroup)
	LeadTimeSourceAccount      = string(constants.LeadTimeSourceAccount)
	LeadTimeSourceManual       = string(constants.LeadTimeSourceManual)
)

// Commitment is what an order promises: the date it is due to ship, how many days that was, and which rule decided.
//
// All three are stamped onto the order at issue rather than derived on read. A lead time is a rule that can be renegotiated; a commitment is a fact about a moment. Deriving the date later would let a customer moving from 30 days to 21 retroactively make last month's orders late, which is the opposite of what a contract is for. Transit is stamped for the same reason: a carrier estimate refreshed next month must not silently move a date the customer already has.
type Commitment struct {
	ShipByDate   time.Time
	LeadTimeDays int
	Source       string
	// TransitDays is the carrier transit subtracted to get ShipByDate, nil when transit was unknown or did not apply.
	TransitDays *int
	// TransitSource names where TransitDays came from, empty when TransitDays is nil.
	TransitSource string
}

// Transit is how long the carrier takes to cover an order's lane, in business days, and where that number came from.
type Transit struct {
	Days   int
	Source string
}

// LeadTimeInput is the chain, gathered once. Nil means "not set at this level, keep looking"; a configured zero is a real answer and means same-day.
type LeadTimeInput struct {
	// CustomerLeadTimeDays is the customer's own commitment, from account_relation.
	CustomerLeadTimeDays *int
	// AccountGroupLeadTimeDays is inherited by every customer in the group that has not set its own.
	AccountGroupLeadTimeDays *int
	// AccountLeadTimeDays is the account-wide default, the last fallback.
	AccountLeadTimeDays *int
}

// ResolveLeadTime picks the number of days and names the rule that produced it.
//
// The chain, most specific first: the customer, then the customer's account group, then the account default. Returns false only when nothing in the chain holds a usable value, which is a misconfiguration rather than a normal state — the account default is not nullable.
//
// A negative value is skipped rather than honoured. It can only arrive from a hand-written database row (the write path rejects it), and a commitment that falls before the order was placed is worse than falling through to the next rule.
func ResolveLeadTime(in LeadTimeInput) (days int, source string, ok bool) {
	for _, candidate := range []struct {
		days   *int
		source string
	}{
		{in.CustomerLeadTimeDays, LeadTimeSourceCustomer},
		{in.AccountGroupLeadTimeDays, LeadTimeSourceAccountGroup},
		{in.AccountLeadTimeDays, LeadTimeSourceAccount},
	} {
		if candidate.days != nil && *candidate.days >= 0 {
			return *candidate.days, candidate.source, true
		}
	}
	return 0, "", false
}

// ResolveCommitment turns an order's issue date into its ship-by commitment.
//
// An explicitly promised date wins the whole chain and stamps source `manual`: someone negotiated this order specifically, and a default must never quietly overwrite that. It is a delivery date, so the order has to leave early enough for the carrier to cover the lane — ship-by is the promised date less transit, and the difference between the two is the whole point of tracking transit at all. Its LeadTimeDays is the span actually committed to, reported even when it is shorter than any configured rule — that difference is the exception worth being able to find later.
//
// A nil transit means the lane was never quoted and the service level carries no default. Ship-by then falls back to the promised date outright, which is how the system behaved before transit existed: a guess at transit would be worse than visibly having none.
//
// The lead-time branch takes no transit. A configured lead time is already a ship lead time — the days a customer waits before the order leaves — so subtracting transit from it would deduct the same journey twice.
//
// Dates are calendar days in UTC, truncated to the day, because ship_by_date is a DATE column and a commitment is a day rather than an instant. Transit alone is counted in business days, matching how carriers quote it.
func ResolveCommitment(issuedAt time.Time, promisedAt *time.Time, in LeadTimeInput, transit *Transit) (Commitment, bool) {
	issueDate := dateOnlyUTC(issuedAt)

	if promisedAt != nil {
		shipBy := dateOnlyUTC(*promisedAt)
		commitment := Commitment{Source: LeadTimeSourceManual}
		if transit != nil {
			shipBy = SubtractBusinessDays(shipBy, transit.Days)
			days := transit.Days
			commitment.TransitDays = &days
			commitment.TransitSource = transit.Source
		}
		commitment.ShipByDate = shipBy
		commitment.LeadTimeDays = int(shipBy.Sub(issueDate).Hours() / 24)
		return commitment, true
	}

	days, source, ok := ResolveLeadTime(in)
	if !ok {
		return Commitment{}, false
	}

	return Commitment{
		ShipByDate:   issueDate.AddDate(0, 0, days),
		LeadTimeDays: days,
		Source:       source,
	}, true
}

func dateOnlyUTC(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
