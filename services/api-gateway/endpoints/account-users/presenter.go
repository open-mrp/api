package accountuserep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func AccountUserPresenter(au *pb.AccountUserDetail) apiresource.AccountUser {
	if au == nil {
		return apiresource.AccountUser{}
	}

	result := apiresource.AccountUser{
		ID:         au.Id,
		Object:     constants.ObjectTypeAccountUser,
		Name:       au.Name,
		Email:      au.Email,
		Username:   au.Username,
		ImageURL:   au.ImageUrl,
		Status:     constants.AccountUserStatus(au.StatusCode),
		LastUsedAt: grpcutil.TimestampToTimePtr(au.LastUsedAt),
		CreatedAt:  grpcutil.TimestampToTime(au.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(au.UpdatedAt),
	}

	if au.RoleId != nil {
		result.Role = &apiresource.Role{
			ID:       *au.RoleId,
			Object:   constants.ObjectTypeRole,
			Name:     deref(au.RoleName),
			TypeCode: constants.RoleTypeCode(deref(au.RoleTypeCode)),
			Owner:    apiresource.SystemOwner(),
		}
	}

	if au.DepartmentId != nil {
		result.Department = &apiresource.Department{
			ID:     *au.DepartmentId,
			Object: constants.ObjectTypeDepartment,
			Name:   deref(au.DepartmentName),
		}
	}

	return result
}

func AccountUserListPresenter(resp *pb.ListAccountUsersResponse) *apiresource.List[apiresource.AccountUser] {
	if resp == nil {
		return apiresource.NewList[apiresource.AccountUser](nil, apiresource.PageInfo{})
	}

	users := make([]apiresource.AccountUser, len(resp.AccountUsers))
	for i, au := range resp.AccountUsers {
		users[i] = AccountUserPresenter(au)
	}

	var pi apiresource.PageInfo
	if resp.PageInfo != nil {
		pi = apiresource.PageInfo{
			NextCursor:  resp.PageInfo.NextCursor,
			PrevCursor:  resp.PageInfo.PrevCursor,
			HasNextPage: resp.PageInfo.HasNextPage,
			HasPrevPage: resp.PageInfo.HasPrevPage,
		}
	}

	return apiresource.NewList(users, pi)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
