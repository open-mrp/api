package scheduling

import (
	"testing"
	"time"
)

// plantMonThu is a Monday-to-Thursday factory with a 3pm pickup, the configuration that motivated calendars.
func plantMonThu(t *testing.T, loc *time.Location, closures ...time.Time) Calendars {
	t.Helper()
	return Calendars{
		Receive:      mustCalendar(t, "1111100"),
		Carrier:      mustCalendar(t, "1111100"),
		Ship:         mustCalendar(t, "1111000", closures...),
		ShipCutoff:   "15:00",
		ShipLocation: loc,
	}
}

func TestResolveCommitment_ShipByPinIsTakenAsGiven(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.August, 4)
	pinned := onDay(2026, time.August, 19) // Wednesday, a day the plant ships

	got, ok := ResolveCommitment(issued, CommitmentBasis{ShipByOverrideDate: &pinned}, LeadTimeInput{AccountLeadTimeDays: new(30)}, &Transit{Days: 3, Source: "carrier_lane"}, plantMonThu(t, time.UTC))
	if !ok {
		t.Fatal("expected a commitment")
	}
	if !got.ShipByDate.Equal(pinned) {
		t.Fatalf("ship-by = %s, want %s", got.ShipByDate.Format(time.DateOnly), pinned.Format(time.DateOnly))
	}
	if got.Source != LeadTimeSourceOrderShipBy {
		t.Fatalf("source = %q, want order_ship_by", got.Source)
	}
	// A pinned ship date is already the day freight leaves, so the journey must not be deducted from it a second time.
	if got.TransitDays != nil {
		t.Fatalf("transit = %v, want nil on a pinned ship date", got.TransitDays)
	}
}

// Pinning a date the plant is shut on resolves to the nearest earlier shipping day rather than standing as an impossibility. The move is reported so it can be shown before the order is saved.
func TestResolveCommitment_ShipByPinStillSnapsToAShippingDay(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.August, 4)
	pinned := onDay(2026, time.August, 22) // Saturday

	got, ok := ResolveCommitment(issued, CommitmentBasis{ShipByOverrideDate: &pinned}, LeadTimeInput{}, nil, plantMonThu(t, time.UTC))
	if !ok {
		t.Fatal("expected a commitment")
	}
	// Saturday the 22nd walks back past Friday, which a Monday-to-Thursday plant also does not ship, to Thursday the 20th.
	if want := onDay(2026, time.August, 20); !got.ShipByDate.Equal(want) {
		t.Fatalf("ship-by = %s, want %s", got.ShipByDate.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	if got.CalendarAdjustmentDays != 2 {
		t.Fatalf("calendar adjustment = %d, want 2", got.CalendarAdjustmentDays)
	}
}

func TestResolveCommitment_LeadTimeOverrideReplacesTheChain(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.August, 4) // Tuesday
	override := 7

	got, ok := ResolveCommitment(issued, CommitmentBasis{LeadTimeOverrideDays: &override}, LeadTimeInput{CustomerLeadTimeDays: new(30), AccountLeadTimeDays: new(45)}, nil, DefaultCalendars())
	if !ok {
		t.Fatal("expected a commitment")
	}
	if want := onDay(2026, time.August, 11); !got.ShipByDate.Equal(want) {
		t.Fatalf("ship-by = %s, want %s", got.ShipByDate.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	if got.LeadTimeDays != 7 || got.Source != LeadTimeSourceOrderLeadTime {
		t.Fatalf("got (%d, %q), want (7, order_lead_time)", got.LeadTimeDays, got.Source)
	}
}

// A same-day override is a real commitment, distinct from having set nothing, and must not fall through to the customer's standing rule.
func TestResolveCommitment_ZeroLeadTimeOverrideIsAnAnswer(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.August, 4)
	override := 0

	got, ok := ResolveCommitment(issued, CommitmentBasis{LeadTimeOverrideDays: &override}, LeadTimeInput{AccountLeadTimeDays: new(30)}, nil, DefaultCalendars())
	if !ok {
		t.Fatal("expected a commitment")
	}
	if !got.ShipByDate.Equal(issued) || got.Source != LeadTimeSourceOrderLeadTime {
		t.Fatalf("got (%s, %q), want (the issue date, order_lead_time)", got.ShipByDate.Format(time.DateOnly), got.Source)
	}
}

// A promise to deliver on a Saturday means the goods have to be on the dock by the Friday. Subtracting transit from the Saturday would buy a day that does not exist.
func TestResolveCommitment_PromisedSaturdaySnapsBackToFriday(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.August, 4)
	promised := onDay(2026, time.August, 22) // Saturday

	cals := Calendars{Receive: mustCalendar(t, "1111100"), Carrier: mustCalendar(t, "1111100"), Ship: mustCalendar(t, "1111100")}

	got, ok := ResolveCommitment(issued, CommitmentBasis{PromisedAt: &promised}, LeadTimeInput{}, &Transit{Days: 3, Source: "carrier_lane"}, cals)
	if !ok {
		t.Fatal("expected a commitment")
	}
	// Delivery snaps to Friday the 21st; three carrier days back is Tuesday the 18th.
	if want := onDay(2026, time.August, 18); !got.ShipByDate.Equal(want) {
		t.Fatalf("ship-by = %s, want %s", got.ShipByDate.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	// One day lost to the dock being shut, none to the shipping calendar. Transit's own contribution is excluded.
	if got.CalendarAdjustmentDays != 1 {
		t.Fatalf("calendar adjustment = %d, want 1", got.CalendarAdjustmentDays)
	}
}

// The two calendars compound: the dock is shut Saturday and the plant does not ship Friday, so both move the date.
func TestResolveCommitment_ReceiveAndShipCalendarsBothApply(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.August, 4)
	promised := onDay(2026, time.August, 22) // Saturday

	got, ok := ResolveCommitment(issued, CommitmentBasis{PromisedAt: &promised}, LeadTimeInput{}, &Transit{Days: 1, Source: "carrier_lane"}, plantMonThu(t, time.UTC))
	if !ok {
		t.Fatal("expected a commitment")
	}
	// Delivery snaps to Friday the 21st, one carrier day back is Thursday the 20th, which the plant does ship.
	if want := onDay(2026, time.August, 20); !got.ShipByDate.Equal(want) {
		t.Fatalf("ship-by = %s, want %s", got.ShipByDate.Format(time.DateOnly), want.Format(time.DateOnly))
	}
}

func TestResolveCommitment_ClosureInsideTheTransitWindowExtendsTheWalk(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.November, 2)
	promised := onDay(2026, time.November, 30) // Monday

	thanksgiving := onDay(2026, time.November, 26)
	cals := Calendars{
		Receive: mustCalendar(t, "1111100", thanksgiving),
		Carrier: mustCalendar(t, "1111100", thanksgiving),
		Ship:    mustCalendar(t, "1111100", thanksgiving),
	}

	got, ok := ResolveCommitment(issued, CommitmentBasis{PromisedAt: &promised}, LeadTimeInput{}, &Transit{Days: 3, Source: "carrier_lane"}, cals)
	if !ok {
		t.Fatal("expected a commitment")
	}
	// Three moving days back from Monday the 30th, with Thanksgiving on the 26th shut, are the 27th, 25th and 24th.
	if want := onDay(2026, time.November, 24); !got.ShipByDate.Equal(want) {
		t.Fatalf("ship-by = %s, want %s", got.ShipByDate.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	// Without the holiday the same lane would have landed on the 25th, so the closure is worth a day.
	noHoliday := Calendars{Receive: mustCalendar(t, "1111100"), Carrier: mustCalendar(t, "1111100"), Ship: mustCalendar(t, "1111100")}
	unaffected, _ := ResolveCommitment(issued, CommitmentBasis{PromisedAt: &promised}, LeadTimeInput{}, &Transit{Days: 3, Source: "carrier_lane"}, noHoliday)
	if !unaffected.ShipByDate.After(got.ShipByDate) {
		t.Fatalf("the holiday did not move the date: %s vs %s", unaffected.ShipByDate.Format(time.DateOnly), got.ShipByDate.Format(time.DateOnly))
	}
}

func TestResolveCommitment_CutoffIsTheShipDateAtThePlantsLocalTime(t *testing.T) {
	t.Parallel()

	eastern := time.FixedZone("EDT", -4*60*60)
	issued := onDay(2026, time.August, 4)
	pinned := onDay(2026, time.August, 19) // Wednesday

	got, ok := ResolveCommitment(issued, CommitmentBasis{ShipByOverrideDate: &pinned}, LeadTimeInput{}, nil, plantMonThu(t, eastern))
	if !ok {
		t.Fatal("expected a commitment")
	}
	if got.ShipByCutoffAt == nil {
		t.Fatal("expected a cutoff instant")
	}
	// 3pm on the 19th in a UTC-4 zone is 19:00 UTC. The date alone says which day freight leaves; this says by when.
	if want := time.Date(2026, time.August, 19, 19, 0, 0, 0, time.UTC); !got.ShipByCutoffAt.UTC().Equal(want) {
		t.Fatalf("cutoff = %s, want %s", got.ShipByCutoffAt.UTC(), want)
	}
}

func TestResolveCommitment_NoCutoffConfiguredLeavesTheInstantUnset(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.August, 4)

	got, ok := ResolveCommitment(issued, CommitmentBasis{}, LeadTimeInput{AccountLeadTimeDays: new(7)}, nil, DefaultCalendars())
	if !ok {
		t.Fatal("expected a commitment")
	}
	if got.ShipByCutoffAt != nil {
		t.Fatalf("cutoff = %v, want nil when the plant has not set one", got.ShipByCutoffAt)
	}
}

// A malformed cutoff must not cost the order its commitment: the date is the promise, the time is a refinement on top of it.
func TestResolveCommitment_MalformedCutoffIsDroppedNotFatal(t *testing.T) {
	t.Parallel()

	cals := DefaultCalendars()
	cals.ShipCutoff = "half past three"

	got, ok := ResolveCommitment(onDay(2026, time.August, 4), CommitmentBasis{}, LeadTimeInput{AccountLeadTimeDays: new(7)}, nil, cals)
	if !ok {
		t.Fatal("expected a commitment despite the bad cutoff")
	}
	if got.ShipByCutoffAt != nil {
		t.Fatal("expected no cutoff instant from an unparseable time")
	}
}

// A promised delivery date is a date, not an instant to be re-read in someone else's zone.
//
// Clients conventionally send a date as midnight UTC. Reading that in the destination zone would make it the previous evening for every customer west of UTC, so an order promised for Monday would be worked back from Sunday and every western customer would silently ship a day early. The pickup cutoff is where a zone genuinely belongs; a delivery date is not.
func TestResolveCommitment_PromisedDateIsAUTCCalendarDay(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.August, 4)
	// Midnight UTC on a Monday, which is Sunday evening across the Americas.
	promised := time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC)

	cals := Calendars{Receive: mustCalendar(t, "1111100"), Carrier: mustCalendar(t, "1111100"), Ship: mustCalendar(t, "1111100")}

	got, ok := ResolveCommitment(issued, CommitmentBasis{PromisedAt: &promised}, LeadTimeInput{}, nil, cals)
	if !ok {
		t.Fatal("expected a commitment")
	}
	if want := onDay(2026, time.August, 24); !got.ShipByDate.Equal(want) {
		t.Fatalf("ship-by = %s, want %s — the promised day must not shift", got.ShipByDate.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	if got.CalendarAdjustmentDays != 0 {
		t.Fatalf("calendar adjustment = %d, want 0 — Monday is a receiving day", got.CalendarAdjustmentDays)
	}
}

// The explanation has to agree with the arithmetic, since both come off the same walk.
func TestResolveCommitment_StepsEndOnTheCommittedDate(t *testing.T) {
	t.Parallel()

	issued := onDay(2026, time.August, 4)
	promised := onDay(2026, time.August, 22)

	got, ok := ResolveCommitment(issued, CommitmentBasis{PromisedAt: &promised}, LeadTimeInput{}, &Transit{Days: 2, Source: "carrier_lane"}, plantMonThu(t, time.UTC))
	if !ok {
		t.Fatal("expected a commitment")
	}
	if len(got.Steps) == 0 {
		t.Fatal("expected a derivation")
	}

	var adjustment int
	for _, step := range got.Steps {
		if step.Code != CommitmentStepCarrierTransit {
			adjustment += step.DaysMoved
		}
	}
	if adjustment != got.CalendarAdjustmentDays {
		t.Fatalf("steps account for %d adjustment days, commitment reports %d", adjustment, got.CalendarAdjustmentDays)
	}

	// The pickup-cutoff step reports the ship date it applies to, so the last dated step is the commitment.
	last := got.Steps[len(got.Steps)-1]
	if !last.Date.Equal(got.ShipByDate) {
		t.Fatalf("last step is %s, ship-by is %s", last.Date.Format(time.DateOnly), got.ShipByDate.Format(time.DateOnly))
	}
}

func TestCommitmentBasis_CountsWhatWasPinned(t *testing.T) {
	t.Parallel()

	when := onDay(2026, time.August, 20)
	days := 7

	if got := (CommitmentBasis{}); !got.IsEmpty() || got.Count() != 0 {
		t.Fatal("an unset basis must read as empty")
	}
	if got := (CommitmentBasis{PromisedAt: &when}); got.IsEmpty() || got.Count() != 1 {
		t.Fatalf("count = %d, want 1", got.Count())
	}
	// The conflict the write path has to reject.
	if got := (CommitmentBasis{PromisedAt: &when, LeadTimeOverrideDays: &days}); got.Count() != 2 {
		t.Fatalf("count = %d, want 2", got.Count())
	}
}

func TestResolveCommitment_IndefinitelyClosedPlantCommitsToNothing(t *testing.T) {
	t.Parallel()

	closures := make([]time.Time, 0, snapBackLimit)
	start := onDay(2026, time.August, 20)
	for i := range snapBackLimit {
		closures = append(closures, start.AddDate(0, 0, -i))
	}
	cals := DefaultCalendars()
	cals.Ship = mustCalendar(t, "1111111", closures...)

	if _, ok := ResolveCommitment(onDay(2026, time.August, 4), CommitmentBasis{ShipByOverrideDate: &start}, LeadTimeInput{}, nil, cals); ok {
		t.Fatal("expected no commitment rather than a date the plant cannot ship on")
	}
}

func TestEstimateArrival_WalksTransitForwardOntoAReceivingDay(t *testing.T) {
	t.Parallel()

	cals := Calendars{
		Receive: mustCalendar(t, "1111100"),
		Carrier: mustCalendar(t, "1111100"),
		Ship:    mustCalendar(t, "1111000"),
	}

	got, ok := EstimateArrival(onDay(2026, time.August, 20), &Transit{Days: 3, Source: LeadTimeSourceManual}, cals)
	if !ok {
		t.Fatal("expected an arrival to resolve")
	}
	if want := onDay(2026, time.August, 25); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
}

func TestEstimateArrival_MovesOntoADayTheDockReceives(t *testing.T) {
	t.Parallel()

	cals := Calendars{
		Receive: mustCalendar(t, "1111000"),
		Carrier: mustCalendar(t, "1111100"),
		Ship:    mustCalendar(t, "1111100"),
	}

	got, ok := EstimateArrival(onDay(2026, time.August, 20), &Transit{Days: 1}, cals)
	if !ok {
		t.Fatal("expected an arrival to resolve")
	}
	if want := onDay(2026, time.August, 24); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
}

func TestEstimateArrival_UnknownTransitHasNoArrival(t *testing.T) {
	t.Parallel()

	if _, ok := EstimateArrival(onDay(2026, time.August, 20), nil, DefaultCalendars()); ok {
		t.Fatal("expected no arrival when transit is unknown")
	}
}
