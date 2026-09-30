package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestInvoicePaymentFlagsFor(t *testing.T) {
	tests := []struct {
		name                string
		invoiced, allocated string
		paid, over          bool
	}{
		{"unpaid", "100", "40", false, false},
		{"paid exactly", "100", "100", true, false},
		{"overpaid", "100", "100.01", true, true},
		{"within half a cent rounds to paid", "100", "99.996", true, false},
		{"half a cent short rounds away from zero", "100", "99.995", false, false},
		{"half a cent over rounds away from zero", "100", "100.005", true, true},
		{"nothing invoiced, nothing paid", "0", "0", true, false},
		{"credit invoice", "-20", "0", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paid, over := InvoicePaymentFlagsFor(d(tt.invoiced), d(tt.allocated))
			if paid != tt.paid || over != tt.over {
				t.Fatalf("InvoicePaymentFlagsFor(%s, %s) = (%v, %v), want (%v, %v)", tt.invoiced, tt.allocated, paid, over, tt.paid, tt.over)
			}
		})
	}
}

func TestTransactionFullyAllocated(t *testing.T) {
	tests := []struct {
		amount, allocated string
		want              bool
	}{
		{"100", "0", false},
		{"100", "99.99", false},
		{"100", "99.995", false}, // 0.005 remains, rounding up to a cent
		{"100", "99.996", true},
		{"100", "99.994", false},
		{"100", "100", true},
		{"100", "150", true},
		{"0", "0", true},
	}
	for _, tt := range tests {
		if got := TransactionFullyAllocated(d(tt.amount), d(tt.allocated)); got != tt.want {
			t.Errorf("TransactionFullyAllocated(%s, %s) = %v, want %v", tt.amount, tt.allocated, got, tt.want)
		}
	}
}
