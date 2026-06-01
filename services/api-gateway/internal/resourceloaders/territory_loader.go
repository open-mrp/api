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

var territoryLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.territory")

func LoadTerritories(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, territoryLoaderTracer, "loader.territories.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetTerritoriesByIDsResponse, error) {
			return coreClient.BatchGetTerritoriesByIDs(ctx, &pb.BatchGetTerritoriesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Territories))
	for _, t := range resp.Territories {
		out[t.Id] = territoryFromProto(t)

		var salesRepID string
		if t.SalesRep != nil {
			salesRepID = t.SalesRep.Id
		}
		meta.Set(constants.ObjectTypeTerritory, t.Id, "sales_rep_id", salesRepID)

		var productLineID string
		if t.ProductLine != nil {
			productLineID = t.ProductLine.Id
		}
		meta.Set(constants.ObjectTypeTerritory, t.Id, "product_line_id", productLineID)
	}
	return out, nil
}

func territoryFromProto(t *pb.TerritoryInfo) *apiresource.Territory {
	return &apiresource.Territory{
		ID:           t.Id,
		Object:       constants.ObjectTypeTerritory,
		State:        t.State,
		StartZipcode: t.StartZipcode,
		EndZipcode:   t.EndZipcode,
		CreatedAt:    grpcutil.TimestampToTime(t.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(t.UpdatedAt),
	}
}
