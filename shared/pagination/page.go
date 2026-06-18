package pagination

import "time"

type PageInfo struct {
	NextCursor  *string
	PrevCursor  *string
	HasNextPage bool
	HasPrevPage bool
}

// BuildPage trims the limit+1 slice, reverses if backward, and computes PageInfo. cursorDir is nil for first-page requests.
func BuildPage[T any](
	items []T,
	limit int32,
	cursorDir *Direction,
	getCreatedAt func(T) time.Time,
	getID func(T) int64,
) ([]T, PageInfo) {
	if len(items) == 0 {
		return items, PageInfo{}
	}

	hasExtra := len(items) > int(limit) // #nosec G115 - len(items) is always <= limit+1, a small pagination value

	if hasExtra {
		items = items[:limit]
	}

	// No cursor — first page
	if cursorDir == nil {
		var pi PageInfo
		pi.HasNextPage = hasExtra
		pi.HasPrevPage = false

		if pi.HasNextPage && len(items) > 0 {
			last := items[len(items)-1]
			nc := EncodeCursor(Cursor{
				CreatedAt: getCreatedAt(last),
				ID:        getID(last),
				Direction: DirectionForward,
			})
			pi.NextCursor = &nc
		}

		return items, pi
	}

	// Forward cursor
	if *cursorDir == DirectionForward {
		var pi PageInfo
		pi.HasNextPage = hasExtra
		pi.HasPrevPage = true

		if pi.HasNextPage && len(items) > 0 {
			last := items[len(items)-1]
			nc := EncodeCursor(Cursor{
				CreatedAt: getCreatedAt(last),
				ID:        getID(last),
				Direction: DirectionForward,
			})
			pi.NextCursor = &nc
		}

		if len(items) > 0 {
			first := items[0]
			pc := EncodeCursor(Cursor{
				CreatedAt: getCreatedAt(first),
				ID:        getID(first),
				Direction: DirectionBackward,
			})
			pi.PrevCursor = &pc
		}

		return items, pi
	}

	// Backward cursor — rows came in ASC order, reverse to DESC
	reverse(items)

	var pi PageInfo
	pi.HasNextPage = true // we came from a next page
	pi.HasPrevPage = hasExtra

	if len(items) > 0 {
		last := items[len(items)-1]
		nc := EncodeCursor(Cursor{
			CreatedAt: getCreatedAt(last),
			ID:        getID(last),
			Direction: DirectionForward,
		})
		pi.NextCursor = &nc
	}

	if pi.HasPrevPage && len(items) > 0 {
		first := items[0]
		pc := EncodeCursor(Cursor{
			CreatedAt: getCreatedAt(first),
			ID:        getID(first),
			Direction: DirectionBackward,
		})
		pi.PrevCursor = &pc
	}

	return items, pi
}

// BuildPageString is like BuildPage but uses string IDs with StringCursor, for tables whose primary key is a varchar.
func BuildPageString[T any](
	items []T,
	limit int32,
	cursorDir *Direction,
	getOccurredAt func(T) time.Time,
	getID func(T) string,
) ([]T, PageInfo) {
	if len(items) == 0 {
		return items, PageInfo{}
	}

	hasExtra := len(items) > int(limit)

	if hasExtra {
		items = items[:limit]
	}

	if cursorDir == nil {
		var pi PageInfo
		pi.HasNextPage = hasExtra

		if pi.HasNextPage && len(items) > 0 {
			last := items[len(items)-1]
			nc := EncodeStringCursor(StringCursor{
				OccurredAt: getOccurredAt(last),
				ID:         getID(last),
				Direction:  DirectionForward,
			})
			pi.NextCursor = &nc
		}

		return items, pi
	}

	if *cursorDir == DirectionForward {
		var pi PageInfo
		pi.HasNextPage = hasExtra
		pi.HasPrevPage = true

		if pi.HasNextPage && len(items) > 0 {
			last := items[len(items)-1]
			nc := EncodeStringCursor(StringCursor{
				OccurredAt: getOccurredAt(last),
				ID:         getID(last),
				Direction:  DirectionForward,
			})
			pi.NextCursor = &nc
		}

		if len(items) > 0 {
			first := items[0]
			pc := EncodeStringCursor(StringCursor{
				OccurredAt: getOccurredAt(first),
				ID:         getID(first),
				Direction:  DirectionBackward,
			})
			pi.PrevCursor = &pc
		}

		return items, pi
	}

	// Backward
	reverse(items)

	var pi PageInfo
	pi.HasNextPage = true
	pi.HasPrevPage = hasExtra

	if len(items) > 0 {
		last := items[len(items)-1]
		nc := EncodeStringCursor(StringCursor{
			OccurredAt: getOccurredAt(last),
			ID:         getID(last),
			Direction:  DirectionForward,
		})
		pi.NextCursor = &nc
	}

	if pi.HasPrevPage && len(items) > 0 {
		first := items[0]
		pc := EncodeStringCursor(StringCursor{
			OccurredAt: getOccurredAt(first),
			ID:         getID(first),
			Direction:  DirectionBackward,
		})
		pi.PrevCursor = &pc
	}

	return items, pi
}

// BuildPageStringWithSearchRank is like BuildPageString but embeds MatchTier in cursors when searchRankEnabled is true (catalog search relevance pagination).
func BuildPageStringWithSearchRank[T any](
	items []T,
	limit int32,
	cursorDir *Direction,
	searchRankEnabled bool,
	getOccurredAt func(T) time.Time,
	getID func(T) string,
	getMatchTier func(T) int32,
) ([]T, PageInfo) {
	if len(items) == 0 {
		return items, PageInfo{}
	}

	hasExtra := len(items) > int(limit)

	if hasExtra {
		items = items[:limit]
	}

	nextStringCursor := func(last T, dir Direction) StringCursor {
		c := StringCursor{
			OccurredAt: getOccurredAt(last),
			ID:         getID(last),
			Direction:  dir,
		}
		if searchRankEnabled {
			t := int(getMatchTier(last))
			c.MatchTier = &t
		}
		return c
	}

	if cursorDir == nil {
		var pi PageInfo
		pi.HasNextPage = hasExtra

		if pi.HasNextPage && len(items) > 0 {
			last := items[len(items)-1]
			nc := EncodeStringCursor(nextStringCursor(last, DirectionForward))
			pi.NextCursor = &nc
		}

		return items, pi
	}

	if *cursorDir == DirectionForward {
		var pi PageInfo
		pi.HasNextPage = hasExtra
		pi.HasPrevPage = true

		if pi.HasNextPage && len(items) > 0 {
			last := items[len(items)-1]
			nc := EncodeStringCursor(nextStringCursor(last, DirectionForward))
			pi.NextCursor = &nc
		}

		if len(items) > 0 {
			first := items[0]
			pc := StringCursor{
				OccurredAt: getOccurredAt(first),
				ID:         getID(first),
				Direction:  DirectionBackward,
			}
			if searchRankEnabled {
				t := int(getMatchTier(first))
				pc.MatchTier = &t
			}
			enc := EncodeStringCursor(pc)
			pi.PrevCursor = &enc
		}

		return items, pi
	}

	reverse(items)

	var pi PageInfo
	pi.HasNextPage = true
	pi.HasPrevPage = hasExtra

	if len(items) > 0 {
		last := items[len(items)-1]
		nc := EncodeStringCursor(nextStringCursor(last, DirectionForward))
		pi.NextCursor = &nc
	}

	if pi.HasPrevPage && len(items) > 0 {
		first := items[0]
		pc := StringCursor{
			OccurredAt: getOccurredAt(first),
			ID:         getID(first),
			Direction:  DirectionBackward,
		}
		if searchRankEnabled {
			t := int(getMatchTier(first))
			pc.MatchTier = &t
		}
		enc := EncodeStringCursor(pc)
		pi.PrevCursor = &enc
	}

	return items, pi
}

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
