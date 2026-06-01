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

var locationLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.location")

func LoadLocations(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, locationLoaderTracer, "loader.locations.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetLocationsByIDsResponse, error) {
			return coreClient.BatchGetLocationsByIDs(ctx, &pb.BatchGetLocationsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Locations))
	for _, loc := range resp.Locations {
		out[loc.Id] = locationFromProto(loc)

		var parentID string
		if loc.ParentId != nil {
			parentID = *loc.ParentId
		}
		meta.Set(constants.ObjectTypeLocation, loc.Id, "parent_id", parentID)

		childIDs := make([]string, len(loc.Children))
		for i, c := range loc.Children {
			childIDs[i] = c.Id
		}
		meta.Set(constants.ObjectTypeLocation, loc.Id, "child_ids", childIDs)
	}
	return out, nil
}

func locationFromProto(loc *pb.LocationInfo) *apiresource.Location {
	return &apiresource.Location{
		ID:        loc.Id,
		Object:    constants.ObjectTypeLocation,
		Name:      loc.Name,
		TypeCode:  constants.LocationTypeCode(loc.TypeCode),
		CreatedAt: grpcutil.TimestampToTime(loc.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(loc.UpdatedAt),
	}
}
