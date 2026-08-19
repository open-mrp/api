package pagination

import (
	apierror "github.com/augno/api/shared/errors"
	sharedpagination "github.com/augno/api/shared/pagination"
	pb "github.com/augno/api/shared/proto/agent"
)

// ParseOptionalStringCursor decodes a platform list cursor for forward-only varchar-ID lists. Empty or omitted cursors are valid for the first page.
func ParseOptionalStringCursor(raw *string) (cursorID string, cursorDir *sharedpagination.Direction, apiErr *apierror.APIError) {
	if raw == nil || *raw == "" {
		return "", nil, nil
	}

	cur, err := sharedpagination.DecodeStringCursor(*raw)
	if err != nil {
		return "", nil, apierror.NewParameterInvalidError("Invalid pagination cursor.", "cursor")
	}
	if cur.Direction == sharedpagination.DirectionBackward {
		return "", nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
	}

	dir := cur.Direction
	return cur.ID, &dir, nil
}

// ToProtoPageInfo maps shared pagination metadata to the agent proto PageInfo type.
func ToProtoPageInfo(pi sharedpagination.PageInfo) *pb.PageInfo {
	return &pb.PageInfo{
		NextCursor:  pi.NextCursor,
		PrevCursor:  pi.PrevCursor,
		HasNextPage: pi.HasNextPage,
		HasPrevPage: pi.HasPrevPage,
	}
}
