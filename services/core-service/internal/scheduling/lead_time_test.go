package scheduling

import (
	"testing"
	"time"
)

func TestResolveLeadTime_PrefersCustomerOverGroupOverAccount(t *testing.T) {
	t.Parallel()

	days, source, ok := ResolveLeadTime(LeadTimeInput{
		CustomerLeadTimeDays:     new(14),
		AccountGroupLeadTimeDays: new(21),
		AccountLeadTimeDays:      new(30),
	})
	if !ok || days != 14 || source != LeadTimeSourceCustomer {
		t.Fatalf("got (%d, %q, %v), want (14, customer, true)", days, source, ok)
	}

	days, source, ok = ResolveLeadTime(LeadTimeInput{
		AccountGroupLeadTimeDays: new(21),
		AccountLeadTimeDays:      new(30),
	})
	if !ok || days != 21 || source != LeadTimeSourceAccountGroup {
		t.Fatalf("got (%d, %q, %v), want (21, account_group, true)", days, source, ok)
	}

	days, source, ok = ResolveLeadTime(LeadTimeInput{AccountLeadTimeDays: new(30)})
	if !ok || days != 30 || source != LeadTimeSourceAccount {
		t.Fatalf("got (%d, %q, %v), want (30, account, true)", days, source, ok)
	}
}

// A configured zero is a real commitment — ship same day — and must not be mistaken for "unset" and fall through to a laxer default.
func TestResolveLeadTime_ZeroIsAnAnswerNotAnAbsence(t *testing.T) {
	t.Parallel()

	days, source, ok := ResolveLeadTime(LeadTimeInput{
		CustomerLeadTimeDays: new(0),
		AccountLeadTimeDays:  new(30),
	})
	if !ok || days != 0 || source != LeadTimeSourceCustomer {
		t.Fatalf("got (%d, %q, %v), want (0, customer, true)", days, source, ok)
	}
}

// A negative can only come from a hand-written row; honouring it would date a commitment before the order existed.
func TestResolveLeadTime_NegativeFallsThrough(t *testing.T) {
	t.Parallel()

	days, source, ok := ResolveLeadTime(LeadTimeInput{
		CustomerLeadTimeDays: new(-5),
		AccountLeadTimeDays:  new(30),
	})
	if !ok || days != 30 || source != LeadTimeSourceAccount {
		t.Fatalf("got (%d, %q, %v), want (30, account, true)", days, source, ok)
	}
}

func TestResolveLeadTime_NothingConfigured(t *testing.T) {
	t.Parallel()

	if _, _, ok := ResolveLeadTime(LeadTimeInput{}); ok {
		t.Fatal("expected no resolution when every level is unset")
	}
}

func TestResolveCommitment_AddsCalendarDaysToTheIssueDate(t *testing.T) {
	t.Parallel()

	issued := time.Date(2026, time.August, 6, 17, 42, 3, 0, time.UTC)

	got, ok := ResolveCommitment(issued, nil, LeadTimeInput{AccountLeadTimeDays: new(30)})
	if !ok {
		t.Fatal("expected a commitment")
	}

	want := time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC)
	if !got.ShipByDate.Equal(want) {
		t.Fatalf("ship-by = %s, want %s", got.ShipByDate, want)
	}
	if got.LeadTimeDays != 30 || got.Source != LeadTimeSourceAccount {
		t.Fatalf("got (%d, %q), want (30, account)", got.LeadTimeDays, got.Source)
	}
}

// The commitment is a day, not an instant: two orders issued at either end of the same day are due on the same date.
func TestResolveCommitment_TruncatesToTheDay(t *testing.T) {
	t.Parallel()

	early := time.Date(2026, time.August, 6, 0, 0, 1, 0, time.UTC)
	late := time.Date(2026, time.August, 6, 23, 59, 59, 0, time.UTC)
	in := LeadTimeInput{AccountLeadTimeDays: new(7)}

	first, _ := ResolveCommitment(early, nil, in)
	second, _ := ResolveCommitment(late, nil, in)

	if !first.ShipByDate.Equal(second.ShipByDate) {
		t.Fatalf("same-day orders got different ship-by dates: %s vs %s", first.ShipByDate, second.ShipByDate)
	}
}

func TestResolveCommitment_PromisedDateWinsEveryRule(t *testing.T) {
	t.Parallel()

	issued := time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)
	promised := time.Date(2026, time.August, 20, 9, 30, 0, 0, time.UTC)

	got, ok := ResolveCommitment(issued, &promised, LeadTimeInput{
		CustomerLeadTimeDays: new(3),
		AccountLeadTimeDays:  new(30),
	})
	if !ok {
		t.Fatal("expected a commitment")
	}
	if got.Source != LeadTimeSourceManual {
		t.Fatalf("source = %q, want manual", got.Source)
	}
	if want := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC); !got.ShipByDate.Equal(want) {
		t.Fatalf("ship-by = %s, want %s", got.ShipByDate, want)
	}
	// The span actually committed to, not the customer's standing rule — that difference is the whole reason this order was negotiated separately.
	if got.LeadTimeDays != 14 {
		t.Fatalf("lead time = %d, want 14", got.LeadTimeDays)
	}
}

// A promise that predates the order is a real (bad) state a rep can create. Recording it truthfully is what lets it be found; silently clamping it to zero would hide an order that is late before it is placed.
func TestResolveCommitment_PromiseBeforeIssueIsRecordedNegative(t *testing.T) {
	t.Parallel()

	issued := time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)
	promised := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

	got, ok := ResolveCommitment(issued, &promised, LeadTimeInput{AccountLeadTimeDays: new(30)})
	if !ok || got.LeadTimeDays != -5 {
		t.Fatalf("got (%d, %v), want (-5, true)", got.LeadTimeDays, ok)
	}
}

// Local-time issue instants must not shift the date: 8pm US Eastern is the next day in UTC, and a commitment that moves with the reader's timezone is not a commitment.
func TestResolveCommitment_NormalizesToUTC(t *testing.T) {
	t.Parallel()

	eastern := time.FixedZone("EST", -5*60*60)
	issued := time.Date(2026, time.August, 6, 20, 0, 0, 0, eastern)

	got, ok := ResolveCommitment(issued, nil, LeadTimeInput{AccountLeadTimeDays: new(1)})
	if !ok {
		t.Fatal("expected a commitment")
	}
	// 2026-08-06 20:00 EST is 2026-08-07 01:00 UTC, so the issue date is the 7th and one day later is the 8th.
	if want := time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC); !got.ShipByDate.Equal(want) {
		t.Fatalf("ship-by = %s, want %s", got.ShipByDate, want)
	}
}

func TestResolveCommitment_NothingConfigured(t *testing.T) {
	t.Parallel()

	issued := time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)
	if _, ok := ResolveCommitment(issued, nil, LeadTimeInput{}); ok {
		t.Fatal("expected no commitment when every level is unset")
	}
}
