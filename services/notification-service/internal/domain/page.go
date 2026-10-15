package domain

import "time"

// PageRequest is a caller's request for one page of a list: the opaque cursor from the previous page, if any, and the page size.
type PageRequest struct {
	Cursor *string
	Limit  int32
}

// KeysetPage asks a repository for the rows after the cursor row in the list's fixed order, at most Limit of them. A nil CursorAt starts from the first row.
type KeysetPage struct {
	CursorAt *time.Time
	CursorID *string
	Limit    int32
}

// Page is one page of a list and the cursor of the next, nil on the last page.
type Page[T any] struct {
	Items      []T
	NextCursor *string
}
