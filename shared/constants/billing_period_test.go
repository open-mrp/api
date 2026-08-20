package constants

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBillingPeriodStart(t *testing.T) {
	t.Parallel()

	end := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), BillingPeriodStart(&end),
		"subscribed accounts count from (period end - 1 month)")

	got := BillingPeriodStart(nil)
	assert.Equal(t, 1, got.Day(), "unsubscribed accounts count from the first of the month")
	assert.Equal(t, time.UTC, got.Location())
}
