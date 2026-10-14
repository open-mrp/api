package apirequest

import (
	"time"

	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreatedRange is the created-date filter every list takes. Embed it in a list request beside object.ListRequest.
type CreatedRange struct {
	// Only records created at or after this time.
	CreatedAfter *time.Time `query:"created_after"`
	// Only records created before this time.
	CreatedBefore *time.Time `query:"created_before"`
}

// Validate rejects a range that ends before it starts, which could only ever return nothing.
func (r CreatedRange) Validate() *apierror.APIError {
	if r.CreatedAfter != nil && r.CreatedBefore != nil && !r.CreatedBefore.After(*r.CreatedAfter) {
		return apierror.NewParameterInvalidError("created_before", "created_before must be later than created_after.")
	}
	return nil
}

// ToProto is the range as list RPCs take it, or nil when neither end is set.
func (r CreatedRange) ToProto() *pb.CreatedRange {
	if r.CreatedAfter == nil && r.CreatedBefore == nil {
		return nil
	}
	out := &pb.CreatedRange{}
	if r.CreatedAfter != nil {
		out.CreatedAfter = timestamppb.New(*r.CreatedAfter)
	}
	if r.CreatedBefore != nil {
		out.CreatedBefore = timestamppb.New(*r.CreatedBefore)
	}
	return out
}
