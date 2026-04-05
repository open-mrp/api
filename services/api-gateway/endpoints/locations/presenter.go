package locationep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func LocationPresenter(sl *pb.LocationInfo) apiresource.Location {
	if sl == nil {
		return apiresource.Location{}
	}

	result := apiresource.Location{
		ID:        sl.Id,
		Object:    constants.ObjectTypeLocation,
		Name:      sl.Name,
		TypeCode:  constants.LocationTypeCode(sl.TypeCode),
		CreatedAt: grpcutil.TimestampToTime(sl.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(sl.UpdatedAt),
	}

	if sl.ParentId != nil && *sl.ParentId != "" {
		parentName := ""
		if sl.ParentName != nil {
			parentName = *sl.ParentName
		}
		var parentTypeCode constants.LocationTypeCode
		if sl.ParentTypeCode != nil {
			parentTypeCode = constants.LocationTypeCode(*sl.ParentTypeCode)
		}
		result.Parent = &apiresource.Location{
			ID:       *sl.ParentId,
			Object:   constants.ObjectTypeLocation,
			Name:     parentName,
			TypeCode: parentTypeCode,
		}
	}

	if sl.Children != nil {
		children := make([]apiresource.Location, len(sl.Children))
		for i, c := range sl.Children {
			children[i] = apiresource.Location{
				ID:       c.Id,
				Object:   constants.ObjectTypeLocation,
				Name:     c.Name,
				TypeCode: constants.LocationTypeCode(c.TypeCode),
			}
		}
		result.Children = apiresource.NewList(children, apiresource.PageInfo{})
	}

	return result
}

func LocationListPresenter(resp *pb.ListLocationsResponse) *apiresource.List[apiresource.Location] {
	if resp == nil {
		return apiresource.NewList[apiresource.Location](nil, apiresource.PageInfo{})
	}

	locations := make([]apiresource.Location, len(resp.Locations))
	for i, sl := range resp.Locations {
		locations[i] = LocationPresenter(sl)
	}

	return apiresource.NewList(locations, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

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

func LocationTypeListPresenter(resp *pb.ListLocationTypesResponse) *apiresource.List[apiresource.LocationType] {
	if resp == nil {
		return apiresource.NewList[apiresource.LocationType](nil, apiresource.PageInfo{})
	}

	types := make([]apiresource.LocationType, len(resp.LocationTypes))
	for i, slt := range resp.LocationTypes {
		types[i] = LocationTypePresenter(slt)
	}

	return apiresource.NewList(types, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
