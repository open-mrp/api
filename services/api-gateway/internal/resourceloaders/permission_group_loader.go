package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var permissionGroupLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.permission_group")

func LoadPermissionGroups(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, permissionGroupLoaderTracer, "loader.permission_groups.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPermissionGroupsByIDsResponse, error) {
			return coreClient.BatchGetPermissionGroupsByIDs(ctx, &pb.BatchGetPermissionGroupsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.PermissionGroups))
	for _, pg := range resp.PermissionGroups {
		out[pg.Id] = permissionGroupFromProto(pg)
	}
	return out, nil
}

func permissionGroupFromProto(pg *pb.PermissionGroupInfo) *apiresource.PermissionGroup {
	perms := make([]apiresource.Permission, len(pg.Permissions))
	for i, p := range pg.Permissions {
		perms[i] = apiresource.Permission{
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

	return &apiresource.PermissionGroup{
		ID:          pg.Id,
		Object:      constants.ObjectTypePermissionGroup,
		Code:        pg.Code,
		Name:        pg.Name,
		Description: pg.Description,
		Permissions: apiresource.NewList(perms, apiresource.PageInfo{}),
		CreatedAt:   grpcutil.TimestampToTime(pg.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(pg.UpdatedAt),
	}
}
