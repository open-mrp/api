package syspropertyep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func SysPropertyPresenter(sp *pb.SysPropertyInfo) apiresource.SysProperty {
	if sp == nil {
		return apiresource.SysProperty{}
	}

	return apiresource.SysProperty{
		ID:     sp.Id,
		Object: constants.ObjectTypeSysProperty,
		Type: &apiresource.SysPropertyType{
			ID:     sp.TypeId,
			Object: constants.ObjectTypeSysPropertyType,
			Name:   sp.TypeName,
			Code:   sp.TypeCode,
		},
		Value:     sp.Value,
		CreatedAt: grpcutil.TimestampToTime(sp.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(sp.UpdatedAt),
	}
}

func SysPropertyListPresenter(ctx context.Context, resp *pb.ListSysPropertiesResponse) *apiresource.List[apiresource.SysProperty] {
	if resp == nil {
		return apiresource.NewList[apiresource.SysProperty](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.SysProperty, len(resp.SysProperties))
	for i, sp := range resp.SysProperties {
		items[i] = SysPropertyPresenter(sp)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
