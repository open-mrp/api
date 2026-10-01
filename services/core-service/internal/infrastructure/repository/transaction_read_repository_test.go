package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func TestTransactionListInOrder(t *testing.T) {
	t.Parallel()

	str := func(s string) *string { return &s }
	now := time.Now()
	tests := []struct {
		name   string
		params domain.ListTransactionsParams
		want   bool
	}{
		{"unfiltered", domain.ListTransactionsParams{}, true},
		{"equality filters", domain.ListTransactionsParams{
			Status: str("allocated"), TypeCodes: []string{"payment"}, MethodCodes: []string{"check"},
			AdjustmentTypeCodes: []string{"write_off"}, CustomerIDs: []string{"ac_1"},
		}, true},
		{"blank search", domain.ListTransactionsParams{Query: str(" ")}, true},
		{"search", domain.ListTransactionsParams{Query: str("TX-1")}, false},
		{"start date", domain.ListTransactionsParams{StartDate: &now}, false},
		{"end date", domain.ListTransactionsParams{EndDate: &now}, false},
		{"customer group", domain.ListTransactionsParams{CustomerGroupIDs: []string{"ag_1"}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := transactionListInOrder(tc.params); got != tc.want {
				t.Errorf("transactionListInOrder = %v, want %v", got, tc.want)
			}
		})
	}
}

// The hint names the indexes, so a rename or a dropped migration turns the list into a 1176 at
// runtime rather than a compile error.
func TestTransactionListIndexes_AreDeclaredInMigrations(t *testing.T) {
	t.Parallel()

	schema := migrationsText(t)
	for _, index := range transactionListIndexes {
		if !strings.Contains(schema, index) {
			t.Errorf("%s is FORCE INDEX'd by the transaction list but no migration creates it", index)
		}
	}
}
