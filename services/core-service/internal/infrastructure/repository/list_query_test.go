package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func TestListFilter_CreatedBetween(t *testing.T) {
	t.Parallel()
	jan := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	f := &listFilter{}
	f.createdBetween("so.created_at", domain.CreatedRange{})
	assert.Empty(t, f.whereSQL(), "an unset range filters nothing")

	f.createdBetween("so.created_at", domain.CreatedRange{After: &jan, Before: &feb})
	assert.Equal(t, "\nWHERE so.created_at >= ?\nAND so.created_at < ?", f.whereSQL())
	assert.Equal(t, []any{jan, feb}, f.args)
}
