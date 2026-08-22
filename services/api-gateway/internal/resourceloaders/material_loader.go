package resourceloaders

import (
	"context"

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

	out := make(map[string]any, len(resp.Materials))
	for _, m := range resp.Materials {
		out[m.Id] = MaterialFromProto(m)
		StashMaterialMeta(ctx, m)
	}
	return out, nil
}

// StashMaterialMeta records the id needed to populate a material's expandable item when the include resolver runs. Pair it with MaterialFromProto so includes work without leaking the nested item.
func StashMaterialMeta(ctx context.Context, m *pb.MaterialInfo) {
	if m == nil {
		return
	}
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeMaterial, m.Id, "item_id", m.ItemId)
}

// MaterialFromProto builds the gated Material resource: the expandable item is left nil and populated only when explicitly requested via the include resolver. Use this — never the full MaterialPresenter — when building a JSON API response.
func MaterialFromProto(m *pb.MaterialInfo) *apiresource.Material {
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
		// Unit left nil: expandable, loaded with real data via ?include=; never fabricated.
	}
}
