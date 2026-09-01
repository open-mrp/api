package service

import (
	"strings"
	"testing"
)

func TestStatementDescriptorSuffix(t *testing.T) {
	tests := []struct {
		name        string
		orderNumber string
		want        string
	}{
		// Stripe rejects an all-digit suffix ("must contain at least one Latin character"), and FormatRecordNumber zero-pads every numeric order number into exactly that.
		{name: "all digits gets a letter prefix", orderNumber: "000123", want: "SO000123"},
		{name: "letters are kept as-is", orderNumber: "SO-000123", want: "SO000123"},
		{name: "punctuation is stripped", orderNumber: "SO#123/A", want: "SO123A"},
		{name: "empty stays empty", orderNumber: "", want: ""},
		{name: "no alphanumerics stays empty", orderNumber: "--/--", want: ""},
		{name: "long numbers truncate to the limit", orderNumber: strings.Repeat("9", 30), want: "SO" + strings.Repeat("9", 20)},
		{name: "long alphanumerics truncate to the limit", orderNumber: "A" + strings.Repeat("9", 30), want: "A" + strings.Repeat("9", 21)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statementDescriptorSuffix(tt.orderNumber)
			if got != tt.want {
				t.Fatalf("statementDescriptorSuffix(%q) = %q, want %q", tt.orderNumber, got, tt.want)
			}
			if len(got) > 22 {
				t.Fatalf("statementDescriptorSuffix(%q) = %q, longer than Stripe's 22-character limit", tt.orderNumber, got)
			}
		})
	}
}
