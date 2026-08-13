package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAckAttachmentFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		orderNumber string
		customerPO  string
		want        string
	}{
		{"no PO", "1042", "", "order-acknowledgement-1042.pdf"},
		{"simple PO", "1042", "PO-8891", "order-acknowledgement-1042-PO-PO-8891.pdf"},
		{"PO with spaces and slashes", "1042", "ACME / 4500 123", "order-acknowledgement-1042-PO-ACME-4500-123.pdf"},
		{"PO with only unsafe characters", "1042", "###", "order-acknowledgement-1042.pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ackAttachmentFilename(tt.orderNumber, tt.customerPO))
		})
	}
}

func TestFilenameSafe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"PO-12345", "PO-12345"},
		{"rev 2.1", "rev-2.1"},
		{"a//b\\\\c", "a-b-c"},
		{"  spaced  out  ", "spaced-out"},
		{"...", ""},
		{"..\\..\\etc", "etc"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, filenameSafe(tt.in), "input %q", tt.in)
	}
}
