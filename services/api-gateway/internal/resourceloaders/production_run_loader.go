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

var productionRunLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.production_run")

// LoadProductionRuns fetches production runs by ID via GetProductionRun and builds reference values with real header data (number, started/completed timestamps) for the sales order's related.production_run. There is no batch RPC, so each ID is fetched individually. The responsible_user FK id is stashed so a nested responsible_user include can resolve.
func LoadProductionRuns(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(ids))
	for _, id := range ids {
		resp, apiErr := grpcutil.CallRPC(ctx, productionRunLoaderTracer, "loader.production_runs.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductionRunResponse, error) {
				return coreProductionRunClient.GetProductionRun(ctx, &pb.GetProductionRunRequest{Id: id}, opts...)
			})
		if apiErr != nil {
			if omitOnUnauthorized(apiErr) {
				return out, nil
			}
			return nil, apiErr
		}
		if resp.ProductionRun == nil {
			continue
		}
		out[resp.ProductionRun.Id] = productionRunReferenceFromProto(resp.ProductionRun)
		if resp.ProductionRun.ResponsibleUserId != "" {
			meta.Set(constants.ObjectTypeProductionRun, resp.ProductionRun.Id, "responsible_user_id", resp.ProductionRun.ResponsibleUserId)
		}
	}
	return out, nil
}

func productionRunReferenceFromProto(info *pb.ProductionRunInfo) *apiresource.ProductionRun {
	return &apiresource.ProductionRun{
		ID:          info.Id,
		Object:      constants.ObjectTypeProductionRun,
		Number:      info.Number,
		BatchCount:  info.BatchCount,
		StartedAt:   grpcutil.TimestampToTimePtr(info.StartedAt),
		CompletedAt: grpcutil.TimestampToTimePtr(info.CompletedAt),
		CreatedAt:   grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(info.UpdatedAt),
	}
}
