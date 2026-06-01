package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var adjustmentTypeLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.adjustment_type")

func LoadAdjustmentTypes(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, adjustmentTypeLoaderTracer, "loader.adjustment_types.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAdjustmentTypesByIDsResponse, error) {
			return coreClient.BatchGetAdjustmentTypesByIDs(ctx, &pb.BatchGetAdjustmentTypesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.AdjustmentTypes))
	for _, at := range resp.AdjustmentTypes {
		out[at.Id] = adjustmentTypeFromProto(at)
	}
	return out, nil
}

func adjustmentTypeFromProto(at *pb.AdjustmentTypeInfo) *apiresource.AdjustmentType {
	return &apiresource.AdjustmentType{
		ID:        at.Id,
		Object:    constants.ObjectTypeAdjustmentType,
		Name:      at.Name,
		Code:      constants.AdjustmentType(at.Code),
		CreatedAt: grpcutil.TimestampToTime(at.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(at.UpdatedAt),
	}
}
