package edidclocationep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func DCLocationPresenter(d *pb.DCLocationProto) apiresource.DCLocation {
	if d == nil {
		return apiresource.DCLocation{}
	}

	loc := apiresource.DCLocation{
		ID:        d.Id,
		Object:    constants.ObjectTypeDCLocation,
		Location:  d.Location,
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}

	if d.CustomerId != "" {
		loc.Customer = &apiresource.DCLocationCustomer{
			ID:     d.CustomerId,
			Object: constants.ObjectTypeCustomer,
			Name:   d.CustomerName,
		}
	}

	return loc
}

func DCLocationListPresenter(resp *pb.ListDCLocationsResponse) *apiresource.List[apiresource.DCLocation] {
	if resp == nil {
		return apiresource.NewList[apiresource.DCLocation](nil, apiresource.PageInfo{})
	}

	locs := make([]apiresource.DCLocation, len(resp.DcLocations))
	for i, d := range resp.DcLocations {
		locs[i] = DCLocationPresenter(d)
	}

	return apiresource.NewList(locs, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
