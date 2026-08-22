package locationep

import (
	"context"

	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func LocationTypePresenter(slt *pb.LocationTypeInfo) apiresource.LocationType {
	if slt == nil {
		return apiresource.LocationType{}
	}

	return apiresource.LocationType{
		ID:        slt.Id,
		Object:    constants.ObjectTypeLocationType,
		Code:      constants.LocationTypeCode(slt.Code),
		Name:      slt.Name,
		CreatedAt: grpcutil.TimestampToTime(slt.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(slt.UpdatedAt),
	}
}

func LocationTypeListPresenter(ctx context.Context, resp *pb.ListLocationTypesResponse) *apiresource.List[apiresource.LocationType] {
	if resp == nil {
		return apiresource.NewList[apiresource.LocationType](nil, apiresource.PageInfo{})
	}

	types := make([]apiresource.LocationType, len(resp.LocationTypes))
	for i, slt := range resp.LocationTypes {
		types[i] = LocationTypePresenter(slt)
	}

	return apiresource.NewList(types, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
