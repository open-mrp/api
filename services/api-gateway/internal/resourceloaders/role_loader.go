package resourceloaders

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var roleLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.role")

func LoadRoles(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, roleLoaderTracer, "loader.roles.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetRolesByIDsResponse, error) {
			return coreClient.BatchGetRolesByIDs(ctx, &pb.BatchGetRolesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Roles))
	for _, r := range resp.Roles {
		out[r.Id] = roleFromProto(r)
		meta.Set(constants.ObjectTypeRole, r.Id, "owner_account_id", r.AccountId)
		perms := formatRolePermissions(r.Permissions)
		meta.Set(constants.ObjectTypeRole, r.Id, "permissions", perms)
	}
	return out, nil
}

func roleFromProto(r *pb.RoleDetail) *apiresource.Role {
	return &apiresource.Role{
		ID:        r.Id,
		Object:    constants.ObjectTypeRole,
		Name:      r.Name,
		TypeCode:  constants.RoleType(r.RoleTypeCode),
		CreatedAt: grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(r.UpdatedAt),
	}
}

func formatRolePermissions(protos []*pb.RolePermissionDetail) []string {
	var perms []string
	for _, rp := range protos {
		if rp == nil {
			continue
		}
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
	}
	return perms
}
