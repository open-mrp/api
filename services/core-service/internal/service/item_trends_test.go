package service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func trendLevel(at time.Time, value string) domain.InventoryLevel {
	return domain.InventoryLevel{At: at, Value: decimal.RequireFromString(value)}
}

// The dashboard chart pairs dates[i] with data[i], so the series is exactly one point per day, each
// the level the day closed on, quiet days carrying the last one forward from the level before the window.
func TestDailyClosingSeries_OnePointPerDayCarriedForwardFromTheOpening(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	day := func(d int, hour int) time.Time { return start.AddDate(0, 0, d).Add(time.Duration(hour) * time.Hour) }

	points := dailyClosingSeries(start, 30, decimal.NewFromInt(40), []domain.InventoryLevel{
		trendLevel(day(2, 23), "12"),
		trendLevel(day(5, 1), "18"),
		trendLevel(day(5, 1), "19"), // same closing instant: the later row (by id) wins
		trendLevel(day(29, 0), "7.5"),
	})

	require.Len(t, points, 30)
	for i, p := range points {
		assert.Equal(t, start.AddDate(0, 0, i), p.Date, "point %d is the start of its UTC day", i)
	}
	assert.Equal(t, "40", points[0].Value, "the series opens at the level logged before the window")
	assert.Equal(t, "40", points[1].Value)
	assert.Equal(t, "12", points[2].Value, "a log just before midnight belongs to its own day")
	assert.Equal(t, "12", points[4].Value, "quiet days carry the last level forward")
	assert.Equal(t, "19", points[5].Value)
	assert.Equal(t, "19", points[28].Value)
	assert.Equal(t, "7.5", points[29].Value, "today closes on today's last log")
}

func TestDailyClosingSeries_NeverLoggedIsZeroThroughout(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	points := dailyClosingSeries(start, 30, decimal.Zero, nil)

	require.Len(t, points, 30)
	for _, p := range points {
		assert.Equal(t, "0", p.Value)
	}
}
