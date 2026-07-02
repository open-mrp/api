package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var accountUserLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.account_user")

func LoadAccountUsers(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountUserLoaderTracer, "loader.account_users.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAccountUsersByIDsResponse, error) {
			return coreClient.BatchGetAccountUsersByIDs(ctx, &pb.BatchGetAccountUsersByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.AccountUsers))
	for _, au := range resp.AccountUsers {
		out[au.Id] = accountUserFromProto(au)

		meta.Set(constants.ObjectTypeAccountUser, au.Id, "user_id", au.UserId)

		var roleID string
		if au.RoleId != nil {
			roleID = *au.RoleId
		}
		meta.Set(constants.ObjectTypeAccountUser, au.Id, "role_id", roleID)

		var departmentID string
		if au.DepartmentId != nil {
			departmentID = *au.DepartmentId
		}
		meta.Set(constants.ObjectTypeAccountUser, au.Id, "department_id", departmentID)
	}
	return out, nil
}

// AccountUserName is a resolved display name + handle (email) + profile photo for an account user,
// plus the assigned role id (for callers that expose ?include=role on actors built from these ids).
type AccountUserName struct {
	Name     *string
	Email    *string
	ImageURL *string
	RoleID   *string
}

// LoadAccountUserNames batch-resolves account_user ids to their display name + email, for hydrating polymorphic actors (e.g. conversation participants) that only carry ids.
func LoadAccountUserNames(ctx context.Context, ids []string) (map[string]AccountUserName, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountUserLoaderTracer, "loader.account_users.names", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAccountUsersByIDsResponse, error) {
			return coreClient.BatchGetAccountUsersByIDs(ctx, &pb.BatchGetAccountUsersByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]AccountUserName, len(resp.AccountUsers))
	for _, au := range resp.AccountUsers {
		out[au.Id] = AccountUserName{Name: au.Name, Email: au.Email, ImageURL: au.ImageUrl, RoleID: au.RoleId}
	}
	return out, nil
}

func accountUserFromProto(au *pb.AccountUserDetail) *apiresource.AccountUser {
	return &apiresource.AccountUser{
		ID:         au.Id,
		Object:     constants.ObjectTypeAccountUser,
		Status:     constants.AccountUserStatus(au.StatusCode),
		LastUsedAt: grpcutil.TimestampToTimePtr(au.LastUsedAt),
		CreatedAt:  grpcutil.TimestampToTime(au.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(au.UpdatedAt),
	}
}
