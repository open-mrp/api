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
		name        string
		params      domain.ListTransactionsParams
		customerIDs []string
		want        []string
		wantCounted []string // indexes of the filters left to count
	}{
		{name: "unfiltered", want: []string{transactionCreatedIndex}},
		{name: "unfiltering status", params: domain.ListTransactionsParams{Status: str("all")}, want: []string{transactionCreatedIndex}},
		{name: "blank search", params: domain.ListTransactionsParams{Query: str(" ")}, want: []string{transactionCreatedIndex}},
		{
			name:   "single-valued filters, never beside created_at",
			params: domain.ListTransactionsParams{Status: str("allocated"), TypeCodes: []string{"payment"}},
			want:   []string{transactionStatusIndex, transactionTypeIndex},
		},
		{name: "one customer", customerIDs: []string{"ac_1"}, want: []string{transactionCustomerIndex}},
		{
			name:        "only multi-valued filters",
			params:      domain.ListTransactionsParams{TypeCodes: []string{"payment", "rebate"}},
			customerIDs: []string{"ac_1", "ac_2"},
			want:        []string{transactionCreatedIndex},
			wantCounted: []string{transactionTypeIndex, transactionCustomerIndex},
		},
		{
			name:        "a multi-valued filter beside single-valued ones",
			params:      domain.ListTransactionsParams{Status: str("unallocated"), TypeCodes: []string{"rebate", "credit_memo"}},
			want:        []string{transactionStatusIndex},
			wantCounted: []string{transactionStatusIndex, transactionTypeIndex},
		},
		{name: "search", params: domain.ListTransactionsParams{Query: str("1001"), StartDate: &now}},
		{name: "funds range", params: domain.ListTransactionsParams{StartDate: &now}, want: []string{transactionFundsIndex}},
		{
			name:        "funds range with a customer",
			params:      domain.ListTransactionsParams{EndDate: &now, Status: str("unallocated"), MethodCodes: []string{"check"}},
			customerIDs: []string{"ac_1", "ac_2"},
			want:        []string{transactionFundsIndex, transactionCustomerIndex},
		},
		{
			name:   "funds range with a common filter",
			params: domain.ListTransactionsParams{StartDate: &now, Status: str("allocated"), TypeCodes: []string{"payment"}},
			want:   []string{transactionFundsIndex},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, counted := transactionListIndexHint(tc.params, tc.customerIDs)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("transactionListIndexHint indexes = %v, want %v", got, tc.want)
			}
			var gotCounted []string
			for _, f := range counted {
				gotCounted = append(gotCounted, f.index)
			}
			if !reflect.DeepEqual(gotCounted, tc.wantCounted) {
				t.Errorf("transactionListIndexHint counted = %v, want %v", gotCounted, tc.wantCounted)
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
