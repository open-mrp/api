package resourceloaders

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
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

	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Roles))
	for _, r := range resp.Roles {
		out[r.Id] = roleFromProto(r)
		meta.Set(constants.ObjectTypeRole, r.Id, "owner_account_id", r.AccountId)
		meta.Set(constants.ObjectTypeRole, r.Id, "permissions", nonNilStrings(r.EffectivePermissions))
	}
	return out, nil
}

func roleFromProto(r *pb.RoleDetail) *apiresource.Role {
	access := types.RoleAccess{}
	for k, v := range r.Access {
		access[constants.PermissionArea(k)] = constants.AccessLevel(v)
	}
	role := &apiresource.Role{
		ID:        r.Id,
		Object:    constants.ObjectTypeRole,
		Name:      r.Name,
		TypeCode:  constants.RoleType(r.RoleTypeCode),
		Access:    apiresource.RoleAccessOf(access),
		Grants:    nonNilStrings(r.Grants),
		Revokes:   nonNilStrings(r.Revokes),
		CreatedAt: grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(r.UpdatedAt),
	}
	if r.TemplateCode != nil {
		role.Template = new(constants.RoleTemplate(*r.TemplateCode))
	}
	return role
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
