package grpc

import (
	"github.com/open-mrp/api/services/core-service/internal/domain"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// createdRangeFromProto is a list RPC's created-date filter; an unset range filters nothing.
func createdRangeFromProto(p *pb.CreatedRange) domain.CreatedRange {
	var r domain.CreatedRange
	if p.GetCreatedAfter() != nil {
		r.After = new(p.GetCreatedAfter().AsTime())
	}
	if p.GetCreatedBefore() != nil {
		r.Before = new(p.GetCreatedBefore().AsTime())
	}
	return r
}
