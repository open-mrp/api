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

var ediRunLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.edi_run")

// LoadEDIRuns fetches EDI runs by ID via BatchGetEDIRunsByIDs. Pure leaf — no nested fields at all.
func LoadEDIRuns(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, ediRunLoaderTracer, "loader.edi_runs.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetEDIRunsByIDsResponse, error) {
			return coreClient.BatchGetEDIRunsByIDs(ctx, &pb.BatchGetEDIRunsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.EdiRuns))
	for _, r := range resp.EdiRuns {
		out[r.Id] = ediRunFromProto(r)
	}
	return out, nil
}

func ediRunFromProto(r *pb.EDIRunProto) *apiresource.EDIRun {
	return &apiresource.EDIRun{
		ID:           r.Id,
		Object:       constants.ObjectTypeEDIRun,
		CompletedAt:  grpcutil.TimestampToTime(r.CompletedAt),
		HasSucceeded: r.HasSucceeded,
		CreatedAt:    grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(r.UpdatedAt),
	}
}
