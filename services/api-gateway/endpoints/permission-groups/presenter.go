package permissiongroupep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func PermissionPresenter(p *pb.PermissionInfo) apiresource.Permission {
	if p == nil {
		return apiresource.Permission{}
	}

	return apiresource.Permission{
		ID:                  p.Id,
		Object:              constants.ObjectTypePermission,
		Code:                p.Code,
		Name:                p.Name,
		Description:         p.Description,
		PermissionGroupCode: p.PermissionGroupCode,
		CreatedAt:           grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt:           grpcutil.TimestampToTime(p.UpdatedAt),
	}
}

func PermissionGroupPresenter(pg *pb.PermissionGroupInfo) apiresource.PermissionGroup {
	if pg == nil {
		return apiresource.PermissionGroup{}
	}

	perms := make([]apiresource.Permission, len(pg.Permissions))
	for i, p := range pg.Permissions {
		perms[i] = PermissionPresenter(p)
	}

	return apiresource.PermissionGroup{
		ID:          pg.Id,
		Object:      constants.ObjectTypePermissionGroup,
		Code:        pg.Code,
		Name:        pg.Name,
		Description: pg.Description,
		Permissions: apiresource.NewList(perms, apiresource.PageInfo{}),
		Owner:       apiresource.SystemOwner(),
		CreatedAt:   grpcutil.TimestampToTime(pg.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(pg.UpdatedAt),
	}
}

func PermissionGroupListPresenter(resp *pb.ListPermissionGroupsResponse) *apiresource.List[apiresource.PermissionGroup] {
	if resp == nil {
		return apiresource.NewList[apiresource.PermissionGroup](nil, apiresource.PageInfo{})
	}

	groups := make([]apiresource.PermissionGroup, len(resp.PermissionGroups))
	for i, pg := range resp.PermissionGroups {
		groups[i] = PermissionGroupPresenter(pg)
	}

	return apiresource.NewList(groups, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
