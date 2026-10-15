package service

import (
	"time"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	apierror "github.com/open-mrp/apikit/apierror"
)

const (
	defaultListPageSize int32 = 25
	maxListPageSize     int32 = 100
)

// keysetPage decodes a page request into the repository's keyset, asking for one row more than the page so the extra row tells whether another page follows.
func keysetPage(req domain.PageRequest) (domain.KeysetPage, int32, *apierror.APIError) {
	at, id, apiErr := decodeCursor(req.Cursor)
	if apiErr != nil {
		return domain.KeysetPage{}, 0, apiErr
	}
	limit := clampLimit(req.Limit, defaultListPageSize, maxListPageSize)
	return domain.KeysetPage{CursorAt: at, CursorID: id, Limit: limit + 1}, limit, nil
}

// finishPage trims the extra row keysetPage asked for and, when it was there, sets the next cursor from the last row kept.
func finishPage[T any](rows []T, limit int32, key func(T) (time.Time, string)) domain.Page[T] {
	page := domain.Page[T]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		at, id := key(page.Items[len(page.Items)-1])
		next := encodeCursor(at, id)
		page.NextCursor = &next
	}
	return page
}
