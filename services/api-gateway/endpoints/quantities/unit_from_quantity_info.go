package quantityep

import (
	unitep "github.com/augno/api/services/api-gateway/endpoints/units"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	pb "github.com/augno/api/shared/proto/core"
)

func UnitFromQuantityInfo(q *pb.QuantityInfo) *apiresource.Unit {
	if q == nil {
		return nil
	}
	if ud := q.GetUnitDetail(); ud != nil {
		u := unitep.UnitPresenter(ud, nil)
		return &u
	}
	return apiresource.ExpandableUnitStub(
		q.UnitId,
		q.UnitName,
		q.UnitAbbreviation,
		q.UnitType,
		grpcutil.TimestampToTime(q.CreatedAt),
	)
}
