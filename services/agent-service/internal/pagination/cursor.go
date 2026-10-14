package pagination

import (
	pb "github.com/open-mrp/api/shared/proto/agent"
	apierror "github.com/open-mrp/apikit/apierror"
	sharedpagination "github.com/open-mrp/apikit/pagination"
)

// ParseOptionalStringCursor decodes a platform list cursor for forward-only varchar-ID lists. Empty or omitted cursors are valid for the first page.
func ParseOptionalStringCursor(raw *string) (cursorID string, cursorDir *sharedpagination.Direction, apiErr *apierror.APIError) {
	if raw == nil || *raw == "" {
		return "", nil, nil
	}

	cur, err := sharedpagination.DecodeStringCursor(*raw)
	if err != nil {
		return "", nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
	}
	if cur.Direction == sharedpagination.DirectionBackward {
		return "", nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
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
