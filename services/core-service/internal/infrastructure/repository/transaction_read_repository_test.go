package repository

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func TestTransactionListIndexHint(t *testing.T) {
	t.Parallel()

	str := func(s string) *string { return &s }
	now := time.Now()
	tests := []struct {
		name   string
		params domain.ListTransactionsParams
		want   []string
	}{
		{"unfiltered", domain.ListTransactionsParams{}, transactionListIndexes},
		{"equality filters", domain.ListTransactionsParams{
			Status: str("allocated"), TypeCodes: []string{"payment"}, CustomerGroupIDs: []string{"ag_1"},
		}, transactionListIndexes},
		{"blank search", domain.ListTransactionsParams{Query: str(" ")}, transactionListIndexes},
		{"search", domain.ListTransactionsParams{Query: str("1001"), StartDate: &now}, nil},
		{"funds range", domain.ListTransactionsParams{StartDate: &now}, []string{transactionFundsIndex}},
		{"funds range with filters", domain.ListTransactionsParams{
			EndDate: &now, Status: str("unallocated"), MethodCodes: []string{"check"}, CustomerGroupIDs: []string{"ag_1"},
		}, []string{transactionFundsIndex, transactionCustomerIndex}},
		{"funds range with a common filter", domain.ListTransactionsParams{
			StartDate: &now, Status: str("allocated"), TypeCodes: []string{"payment"},
		}, []string{transactionFundsIndex}},
		{"funds range, unfiltering status", domain.ListTransactionsParams{StartDate: &now, Status: str("all")}, []string{transactionFundsIndex}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := transactionListIndexHint(tc.params); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("transactionListIndexHint = %v, want %v", got, tc.want)
			}
		})
	}
}

// The hint names the indexes, so a rename or a dropped migration turns the list into a 1176 at
// runtime rather than a compile error.
func TestTransactionListIndexes_AreDeclaredInMigrations(t *testing.T) {
	t.Parallel()

	schema := migrationsText(t)
	for _, index := range append(transactionListIndexes, transactionFundsIndex) {
		if !strings.Contains(schema, index) {
			t.Errorf("%s is FORCE INDEX'd by the transaction list but no migration creates it", index)
		}
	}
}

func TestAccountTransactionIndexHint(t *testing.T) {
	t.Parallel()

	str := func(s string) *string { return &s }
	tests := []struct {
		name   string
		params domain.ListAccountTransactionsParams
		want   []string
	}{
		{"customer only", domain.ListAccountTransactionsParams{}, []string{transactionCustomerIndex}},
		{"status and type", domain.ListAccountTransactionsParams{Status: str("unallocated"), Type: str("payment")},
			[]string{transactionCustomerIndex, transactionStatusIndex, transactionTypeIndex}},
		{"unknown status", domain.ListAccountTransactionsParams{Status: str("all")}, []string{transactionCustomerIndex}},
		{"blank search", domain.ListAccountTransactionsParams{Query: str(" ")}, []string{transactionCustomerIndex}},
		{"search", domain.ListAccountTransactionsParams{Query: str("1001"), Type: str("payment")}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := accountTransactionIndexHint(tc.params); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("accountTransactionIndexHint = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFinanceListIndexes_AreDeclaredInMigrations(t *testing.T) {
	t.Parallel()

	schema := migrationsText(t)
	for _, index := range []string{
		allocationCreatedIndex, allocationTypeIndex, allocationTransactionIndex, allocationInvoiceIndex,
	} {
		if !strings.Contains(schema, index) {
			t.Errorf("%s is FORCE INDEX'd by a finance list but no migration creates it", index)
		}
	}
}
