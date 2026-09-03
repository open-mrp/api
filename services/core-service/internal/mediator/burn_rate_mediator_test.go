package mediator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBurnRateTimeSpanDays(t *testing.T) {
	t.Parallel()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	assert.Equal(t, 1.0, burnRateTimeSpanDays(base, base))
	assert.InDelta(t, 2.0, burnRateTimeSpanDays(base, base.Add(48*time.Hour)), 0.001)
	assert.InDelta(t, 0.5, burnRateTimeSpanDays(base, base.Add(12*time.Hour)), 0.001)
}
