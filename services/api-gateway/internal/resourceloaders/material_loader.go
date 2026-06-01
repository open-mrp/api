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

var materialLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.material")

func LoadMaterials(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, materialLoaderTracer, "loader.materials.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetMaterialsByIDsResponse, error) {
			return coreClient.BatchGetMaterialsByIDs(ctx, &pb.BatchGetMaterialsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Materials))
	for _, m := range resp.Materials {
		out[m.Id] = materialFromProto(m)
		meta.Set(constants.ObjectTypeMaterial, m.Id, "item_id", m.ItemId)
	}
	return out, nil
}

func materialFromProto(m *pb.MaterialInfo) *apiresource.Material {
	return &apiresource.Material{
		ID:         m.Id,
		Object:     constants.ObjectTypeMaterial,
		OrderPoint: materialQuantityFromProto(m.OrderPoint),
		LeadTime:   materialQuantityFromProto(m.LeadTime),
		CreatedAt:  grpcutil.TimestampToTime(m.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(m.UpdatedAt),
	}
}

func materialQuantityFromProto(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	normalized := apiresource.NormalizeQuantityValue(q.Value, q.UnitType)
	if normalized == "0" {
		return nil
	}
	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        normalized,
		DisplayValue: apiresource.FormatDisplayValue(normalized, q.UnitAbbreviation, q.UnitType),
		Unit: apiresource.ExpandableUnitStub(
			q.UnitId,
			q.UnitName,
			q.UnitAbbreviation,
			q.UnitType,
			grpcutil.TimestampToTime(q.CreatedAt),
		),
	}
}
