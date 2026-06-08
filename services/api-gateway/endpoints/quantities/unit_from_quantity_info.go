package quantityep

import (
	unitep "github.com/augno/api/services/api-gateway/endpoints/units"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	pb "github.com/augno/api/shared/proto/core"
)

// UnitFromQuantityInfo returns the fully-resolved Unit when the proto carries
// complete unit detail, or nil otherwise. It never fabricates a placeholder —
// when only a unit id is available, callers stash the id so the unit is loaded
// with real data via LoadUnits when ?include= is requested.
func UnitFromQuantityInfo(q *pb.QuantityInfo) *apiresource.Unit {
	if q == nil {
		return nil
	}
	if ud := q.GetUnitDetail(); ud != nil {
		u := unitep.UnitPresenter(ud, nil)
		return &u
	}
	return nil
}
