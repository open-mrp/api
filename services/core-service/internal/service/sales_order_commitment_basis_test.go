package service

import (
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/field"
)

func TestValidateCommitmentBasisExclusive_AcceptsAtMostOne(t *testing.T) {
	t.Parallel()

	when := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	days := int32(21)

	for _, c := range []struct {
		name      string
		promised  *time.Time
		leadTime  *int32
		shipBy    *time.Time
		wantError bool
	}{
		{"nothing pinned falls through to the customer chain", nil, nil, nil, false},
		{"a promised delivery date alone", &when, nil, nil, false},
		{"a lead-time override alone", nil, &days, nil, false},
		{"a pinned ship date alone", nil, nil, &when, false},
		{"a delivery date and a lead time", &when, &days, nil, true},
		{"a delivery date and a pinned ship date", &when, nil, &when, true},
		{"a lead time and a pinned ship date", nil, &days, &when, true},
		{"all three", &when, &days, &when, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			apiErr := validateCommitmentBasisExclusive(c.promised, c.leadTime, c.shipBy)
			if c.wantError && apiErr == nil {
				t.Fatal("expected a conflict to be rejected")
			}
			if !c.wantError && apiErr != nil {
				t.Fatalf("unexpected error: %s", apiErr.PublicMessage)
			}
		})
	}
}

// The message has to name the fields in conflict, since the caller's next move is to clear one of them.
func TestValidateCommitmentBasisExclusive_NamesTheConflictingFields(t *testing.T) {
	t.Parallel()

	when := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	days := int32(21)

	apiErr := validateCommitmentBasisExclusive(&when, &days, nil)
	if apiErr == nil {
		t.Fatal("expected a conflict")
	}
	for _, want := range []string{"promised_at", "lead_time_override_days"} {
		if !strings.Contains(apiErr.PublicMessage, want) {
			t.Errorf("message %q does not name %q", apiErr.PublicMessage, want)
		}
	}
	if strings.Contains(apiErr.PublicMessage, "ship_by_override_date") {
		t.Errorf("message %q names a field that was not set", apiErr.PublicMessage)
	}
}

func TestSalesOrderCommitmentBasisChanged_IgnoresOmittedFields(t *testing.T) {
	t.Parallel()

	when := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	existingDays := 30
	existing := &domain.SalesOrder{PromisedAt: &when, LeadTimeOverrideDays: &existingDays}

	// An update that touches neither must not re-stamp: re-deriving on every unrelated edit is what makes a commitment stop being a fact about a moment.
	if salesOrderCommitmentBasisChanged(existing, domain.UpdateSalesOrderParams{}) {
		t.Fatal("an update naming no basis must not count as a change")
	}
}

func TestSalesOrderCommitmentBasisChanged_DetectsEveryBasis(t *testing.T) {
	t.Parallel()

	when := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC)
	thirty := 30

	for _, c := range []struct {
		name     string
		existing *domain.SalesOrder
		params   domain.UpdateSalesOrderParams
		want     bool
	}{
		{
			"moving the promised date",
			&domain.SalesOrder{PromisedAt: &when},
			domain.UpdateSalesOrderParams{PromisedAt: field.Set(later)},
			true,
		},
		{
			"setting the same promised date again",
			&domain.SalesOrder{PromisedAt: &when},
			domain.UpdateSalesOrderParams{PromisedAt: field.Set(when)},
			false,
		},
		{
			"clearing a lead-time override",
			&domain.SalesOrder{LeadTimeOverrideDays: &thirty},
			domain.UpdateSalesOrderParams{LeadTimeOverrideDays: field.Clear[int32]()},
			true,
		},
		{
			"clearing a lead-time override that was never set",
			&domain.SalesOrder{},
			domain.UpdateSalesOrderParams{LeadTimeOverrideDays: field.Clear[int32]()},
			false,
		},
		{
			"changing a lead-time override",
			&domain.SalesOrder{LeadTimeOverrideDays: &thirty},
			domain.UpdateSalesOrderParams{LeadTimeOverrideDays: field.Set(int32(21))},
			true,
		},
		{
			"pinning a ship date for the first time",
			&domain.SalesOrder{},
			domain.UpdateSalesOrderParams{ShipByOverrideDate: field.Set(when)},
			true,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := salesOrderCommitmentBasisChanged(c.existing, c.params); got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}
