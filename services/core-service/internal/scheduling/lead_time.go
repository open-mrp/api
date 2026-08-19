package scheduling

import (
	"time"

	"github.com/augno/api/shared/constants"
)

// Lead-time sources, aliased from the shared enum so the engine cannot drift from the API contract.
const (
	LeadTimeSourceCustomer      = string(constants.LeadTimeSourceCustomer)
	LeadTimeSourceAccountGroup  = string(constants.LeadTimeSourceAccountGroup)
	LeadTimeSourceAccount       = string(constants.LeadTimeSourceAccount)
	LeadTimeSourceManual        = string(constants.LeadTimeSourceManual)
	LeadTimeSourceOrderLeadTime = string(constants.LeadTimeSourceOrderLeadTime)
	LeadTimeSourceOrderShipBy   = string(constants.LeadTimeSourceOrderShipBy)
)

// Commitment step codes, naming which rule moved a date. Aliased for the same reason as the sources.
const (
	CommitmentStepBasis           = string(constants.CommitmentStepBasis)
	CommitmentStepReceiveCalendar = string(constants.CommitmentStepReceiveCalendar)
	CommitmentStepCarrierTransit  = string(constants.CommitmentStepCarrierTransit)
	CommitmentStepShipCalendar    = string(constants.CommitmentStepShipCalendar)
	CommitmentStepPickupCutoff    = string(constants.CommitmentStepPickupCutoff)
)

// Commitment is what an order promises: the date it is due to ship, how many days that was, and which rule decided.
//
// All of it is stamped onto the order at issue rather than derived on read. A lead time is a rule that can be renegotiated; a commitment is a fact about a moment. Deriving the date later would let a customer moving from 30 days to 21 retroactively make last month's orders late, which is the opposite of what a contract is for. Transit and the calendar adjustment are stamped for the same reason: a carrier estimate refreshed next month, or a holiday added to a calendar in March, must not silently move a date the customer already has.
type Commitment struct {
	ShipByDate   time.Time
	LeadTimeDays int
	Source       string
	// TransitDays is the carrier transit subtracted to get ShipByDate, nil when transit was unknown or did not apply.
	TransitDays *int
	// TransitSource names where TransitDays came from, empty when TransitDays is nil.
	TransitSource string
	// ShipByCutoffAt is ShipByDate at the plant's pickup cutoff, as an instant. The date alone says which day freight has to leave; this says by when, which is the deadline a shop floor actually works to. Nil when the ship calendar carries no cutoff.
	ShipByCutoffAt *time.Time
	// CalendarAdjustmentDays is how many days the receiving and shipping calendars pulled ShipByDate back beyond what transit alone accounted for. Zero when every date already landed on an open day, and the one number that answers "why the 24th and not the 27th".
	CalendarAdjustmentDays int
	// Steps is the ordered derivation, one entry per rule that touched the date. Built on the way through so a preview and the stamped commitment are produced by the same walk and cannot disagree about how a date was reached.
	Steps []CommitmentStep
}

// CommitmentStep is one rule's contribution to a ship-by date.
type CommitmentStep struct {
	// Code names the rule that applied.
	Code string
	// Date is where the running date stood after this rule.
	Date time.Time
	// DaysMoved is how far this rule pulled the date back. Zero means the rule applied and changed nothing, which is worth reporting: it says the promised date was already a day the customer receives.
	DaysMoved int
	// Detail carries the rule's own parameter — the transit days counted, the cutoff time applied — for a caller rendering an explanation.
	Detail string
}

// Transit is how long the carrier takes to cover an order's lane, in the days it moves freight, and where that number came from.
type Transit struct {
	Days   int
	Source string
}

// CommitmentBasis is the explicit input somebody pinned on one order, displacing the standing lead-time chain.
//
// At most one field may be set. The three are alternative answers to the same question and combining them has no meaning — a delivery date and a ship date cannot both be the thing being promised — so the service layer rejects more than one rather than inventing a precedence nobody could predict. The ordering ResolveCommitment applies is a defence against a bad row, not a documented rule.
type CommitmentBasis struct {
	// PromisedAt is a promised *delivery* date. The order has to leave early enough for the carrier to cover the lane, so this is the only basis transit is subtracted from.
	PromisedAt *time.Time
	// LeadTimeOverrideDays is this order's own lead time in days, replacing whatever the customer chain would have resolved to.
	LeadTimeOverrideDays *int
	// ShipByOverrideDate pins the ship date itself, bypassing transit and the receiving calendar.
	ShipByOverrideDate *time.Time
}

// IsEmpty reports whether nothing was pinned, so the standing chain applies.
func (b CommitmentBasis) IsEmpty() bool {
	return b.PromisedAt == nil && b.LeadTimeOverrideDays == nil && b.ShipByOverrideDate == nil
}

// Count is how many bases were pinned. Anything above one is a conflict the caller must reject.
func (b CommitmentBasis) Count() int {
	n := 0
	for _, set := range []bool{b.PromisedAt != nil, b.LeadTimeOverrideDays != nil, b.ShipByOverrideDate != nil} {
		if set {
			n++
		}
	}
	return n
}

// Calendars are the day-sets a commitment has to respect, plus the zones its two time-of-day boundaries are read in.
//
// Three separate calendars because the three parties genuinely differ: a plant may tender freight Monday to Thursday, its carrier moves Monday to Friday, and a customer's dock has its own days and its own holidays. Collapsing them into one weekday rule is what produced ship-by dates nobody could ship on.
type Calendars struct {
	// Receive is the days the customer's dock accepts freight.
	Receive Calendar
	// Carrier is the days the carrier moves freight. Transit is counted in these.
	Carrier Calendar
	// Ship is the days the plant tenders freight.
	Ship Calendar
	// ShipCutoff is the local time freight has to be tendered by, as "15:00". Empty when the plant has not set one, in which case a ship-by date carries no time of day.
	ShipCutoff string
	// ShipLocation is the zone ShipCutoff is read in. Nil falls back to UTC.
	ShipLocation *time.Location
}

// DefaultCalendars is Monday to Friday for all three parties with no closures and no cutoff, which is how every date was computed before calendars existed.
func DefaultCalendars() Calendars {
	return Calendars{Receive: DefaultCalendar(), Carrier: DefaultCalendar(), Ship: DefaultCalendar()}
}

// LeadTimeInput is the standing chain, gathered once. Nil means "not set at this level, keep looking"; a configured zero is a real answer and means same-day.
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
// Four bases, most specific first. A pinned ship date is taken as given. A promised delivery date has the customer's receiving days, the carrier's transit and the plant's shipping days worked back through it, because the difference between when a customer wants it and when it has to leave is the whole point of tracking transit. A per-order lead time replaces the standing chain. Failing all three, the chain itself decides.
//
// Only the promised-delivery branch subtracts transit. Every other basis already names a *ship* date or a *ship* lead time — the days a customer waits before the order leaves — so deducting the journey again would charge for it twice.
//
// Every branch ends on a day the plant actually ships. A ship-by date on a closed Friday or inside a shutdown week is not a deadline anybody can meet, and leaving it there is what made orders late on the day they were created.
//
// Dates are calendar days truncated to the day, because ship_by_date is a DATE column and a commitment is a day rather than an instant. The one exception is ShipByCutoffAt, which exists precisely to name a time.
//
// Returns false when no rule produced a date, or when a calendar is closed indefinitely. Leaving the commitment unstamped is the honest outcome: an order with no ship-by date reads as uncommitted, where a fabricated one would read as a promise nobody made.
func ResolveCommitment(issuedAt time.Time, basis CommitmentBasis, in LeadTimeInput, transit *Transit, cals Calendars) (Commitment, bool) {
	issueDate := dateOnlyUTC(issuedAt)

	switch {
	case basis.ShipByOverrideDate != nil:
		return finishCommitment(issueDate, *basis.ShipByOverrideDate, LeadTimeSourceOrderShipBy, nil, cals, nil)

	case basis.PromisedAt != nil:
		return resolvePromisedCommitment(issueDate, *basis.PromisedAt, transit, cals)

	case basis.LeadTimeOverrideDays != nil && *basis.LeadTimeOverrideDays >= 0:
		return finishCommitment(issueDate, issueDate.AddDate(0, 0, *basis.LeadTimeOverrideDays), LeadTimeSourceOrderLeadTime, nil, cals, nil)

	default:
		days, source, ok := ResolveLeadTime(in)
		if !ok {
			return Commitment{}, false
		}
		return finishCommitment(issueDate, issueDate.AddDate(0, 0, days), source, nil, cals, nil)
	}
}

// resolvePromisedCommitment works a promised delivery date back to the day the order has to leave.
//
// The receiving calendar is applied before transit, not after. A promise to deliver on a Saturday means the goods have to be on the dock by the Friday, and subtracting transit from the Saturday would buy a day that does not exist. Doing it in this order also makes the two adjustments separately visible, so the explanation can say which one moved the date.
//
// A nil transit means the lane was never quoted and the service level carries no default. The journey is then treated as instant, which is how the system behaved before transit existed: a guess would be worse than visibly having none.
func resolvePromisedCommitment(issueDate, promisedAt time.Time, transit *Transit, cals Calendars) (Commitment, bool) {
	// Read as a UTC calendar day, not re-interpreted in the customer's zone.
	//
	// A promised delivery date is a date, and the conventional way to send one is midnight UTC. Reading that instant in the destination zone would make it the previous evening for every customer west of UTC — so an order promised for Monday would be worked back from Sunday, and every western customer would silently ship a day early. The zone still decides the pickup cutoff, which is a real time of day; a delivery date is not.
	arrival := dateOnlyUTC(promisedAt)
	steps := []CommitmentStep{{Code: CommitmentStepBasis, Date: arrival}}

	arrival, receiveMoved, ok := cals.Receive.SnapBack(arrival)
	if !ok {
		return Commitment{}, false
	}
	steps = append(steps, CommitmentStep{Code: CommitmentStepReceiveCalendar, Date: arrival, DaysMoved: receiveMoved})

	shipBy := arrival
	if transit != nil {
		shipBy, ok = cals.Carrier.SubtractDays(arrival, transit.Days)
		if !ok {
			return Commitment{}, false
		}
		steps = append(steps, CommitmentStep{Code: CommitmentStepCarrierTransit, Date: shipBy, DaysMoved: daysBetween(arrival, shipBy), Detail: transit.Source})
	}

	return finishCommitment(issueDate, shipBy, LeadTimeSourceManual, transit, cals, steps)
}

// finishCommitment snaps a candidate ship date onto a day the plant ships, stamps the pickup cutoff, and assembles the record.
//
// Shared by every branch so the guarantee that a ship-by date is always shippable holds however the date was arrived at, including a date somebody pinned by hand. Somebody who pins an impossible date gets the nearest earlier open day rather than an impossibility; the move is reported in Steps so it can be shown before the order is saved rather than discovered after it is issued.
func finishCommitment(issueDate, candidate time.Time, source string, transit *Transit, cals Calendars, steps []CommitmentStep) (Commitment, bool) {
	candidate = dateOnlyUTC(candidate)
	if len(steps) == 0 {
		steps = []CommitmentStep{{Code: CommitmentStepBasis, Date: candidate}}
	}

	shipBy, shipMoved, ok := cals.Ship.SnapBack(candidate)
	if !ok {
		return Commitment{}, false
	}
	steps = append(steps, CommitmentStep{Code: CommitmentStepShipCalendar, Date: shipBy, DaysMoved: shipMoved})

	commitment := Commitment{
		ShipByDate:   shipBy,
		LeadTimeDays: daysBetween(issueDate, shipBy),
		Source:       source,
		Steps:        steps,
	}

	if transit != nil {
		days := transit.Days
		commitment.TransitDays = &days
		commitment.TransitSource = transit.Source
	}

	// The adjustment is what the calendars cost beyond the journey itself, which is why transit's own contribution is excluded: a caller comparing ship-by against transit_days wants to know what the remaining gap was spent on.
	for _, step := range steps {
		if step.Code != CommitmentStepCarrierTransit {
			commitment.CalendarAdjustmentDays += step.DaysMoved
		}
	}

	if cutoff, ok := applyCutoff(shipBy, cals.ShipCutoff, cals.ShipLocation); ok {
		commitment.ShipByCutoffAt = &cutoff
		commitment.Steps = append(commitment.Steps, CommitmentStep{Code: CommitmentStepPickupCutoff, Date: shipBy, Detail: cals.ShipCutoff})
	}

	return commitment, true
}

// applyCutoff places a ship-by date at the plant's cutoff time in the plant's own zone, returned as an instant.
//
// A malformed cutoff is dropped rather than failing the commitment. The date is the load-bearing part of the promise and the time is an operational refinement on top of it; refusing to commit an order because somebody typed a bad time into settings would be the wrong trade.
func applyCutoff(shipBy time.Time, cutoff string, loc *time.Location) (time.Time, bool) {
	if cutoff == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("15:04", cutoff)
	if err != nil {
		return time.Time{}, false
	}
	if loc == nil {
		loc = time.UTC
	}
	return time.Date(shipBy.Year(), shipBy.Month(), shipBy.Day(), parsed.Hour(), parsed.Minute(), 0, 0, loc), true
}

// daysBetween counts whole days from one date to another, negative when to precedes from.
func daysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}

func dateOnlyUTC(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
