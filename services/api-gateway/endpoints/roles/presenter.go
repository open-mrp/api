package roleep

import (
	"fmt"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func RolePermissionPresenter(rp *pb.RolePermissionDetail) []string {
	if rp == nil {
		return nil
	}

	perms := make([]string, 0, 4)
	if rp.Create {
		perms = append(perms, fmt.Sprintf("%s:create", rp.PermissionCode))
	}
	if rp.Read {
		perms = append(perms, fmt.Sprintf("%s:read", rp.PermissionCode))
	}
	if rp.Update {
		perms = append(perms, fmt.Sprintf("%s:update", rp.PermissionCode))
	}
	if rp.Delete {
		perms = append(perms, fmt.Sprintf("%s:delete", rp.PermissionCode))
	}

	return perms
}

func RolePresenter(r *pb.RoleDetail, ownerAccount *apiresource.Account) apiresource.Role {
	if r == nil {
		return apiresource.Role{}
	}

	role := apiresource.Role{
		ID:        r.Id,
		Object:    constants.ObjectTypeRole,
		Name:      r.Name,
		TypeCode:  constants.RoleTypeCode(r.RoleTypeCode),
		Owner:     apiresource.NewOwnerWithAccount(stringPtrIfNotEmpty(r.AccountId), ownerAccount),
		CreatedAt: grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(r.UpdatedAt),
	}

	if len(r.Permissions) > 0 {
		perms := make([]string, 0, len(r.Permissions))
		for _, rp := range r.Permissions {
			if rp != nil {
				perms = append(perms, RolePermissionPresenter(rp)...)
			}
		}
		role.Permissions = &perms
	}

	return role
}

func RoleListPresenter(resp *pb.ListRolesResponse, ownerAccount *apiresource.Account) *apiresource.List[apiresource.Role] {
	if resp == nil {
		return apiresource.NewList[apiresource.Role](nil, apiresource.PageInfo{})
	}

	roles := make([]apiresource.Role, len(resp.Roles))
	for i, r := range resp.Roles {
		roles[i] = RolePresenter(r, ownerAccount)
	}

	return apiresource.NewList(roles, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
