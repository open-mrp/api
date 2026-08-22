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

var dcLocationLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.dc_location")

// LoadDCLocations fetches DC locations by ID via BatchGetDCLocationsByIDs. The inline DCLocationCustomer is materialized from the denormalized proto customer_id/customer_name pair — not via an expandable SubField.
func LoadDCLocations(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, dcLocationLoaderTracer, "loader.dc_locations.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetDCLocationsByIDsResponse, error) {
			return coreClient.BatchGetDCLocationsByIDs(ctx, &pb.BatchGetDCLocationsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.DcLocations))
	for _, d := range resp.DcLocations {
		out[d.Id] = dcLocationFromProto(d)
	}
	return out, nil
}

func dcLocationFromProto(d *pb.DCLocationProto) *apiresource.DCLocation {
	return &apiresource.DCLocation{
		ID:       d.Id,
		Object:   constants.ObjectTypeDCLocation,
		Location: d.Location,
		Customer: &apiresource.DCLocationCustomer{
			ID:     d.CustomerId,
			Object: constants.ObjectTypeCustomer,
			Name:   d.CustomerName,
		},
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}
}
