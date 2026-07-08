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

var shipmentLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.shipment")

// LoadShipments fetches shipments by ID via GetShipment and builds expandable Shipment references with real header data. There is no batch RPC for shipments, so each ID is fetched individually. Nested sub-resources (lines, freight, sales_order, …) are their own expandable relations and are not populated here.
func LoadShipments(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(ids))
	for _, id := range ids {
		resp, apiErr := grpcutil.CallRPC(ctx, shipmentLoaderTracer, "loader.shipments.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShipmentResponse, error) {
				return coreShippingClient.GetShipment(ctx, &pb.GetShipmentRequest{Id: id}, opts...)
			})
		if apiErr != nil {
			if omitOnUnauthorized(apiErr) {
				return out, nil
			}
			return nil, apiErr
		}
		if resp.Shipment == nil {
			continue
		}
		out[resp.Shipment.Id] = shipmentReferenceFromProto(resp.Shipment)
	}
	return out, nil
}

func shipmentReferenceFromProto(s *pb.ShipmentInfo) *apiresource.Shipment {
	return &apiresource.Shipment{
		ID:                   s.Id,
		Object:               constants.ObjectTypeShipment,
		Number:               s.Number,
		Note:                 s.Note,
		BillOfLading:         s.BillOfLading,
		MasterTrackingNumber: s.MasterTrackingNumber,
		Status:               constants.ShipmentStatus(s.StatusCode),
		ShippedAt:            grpcutil.TimestampToTimePtr(s.ShippedAt),
		CreatedAt:            grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(s.UpdatedAt),
	}
}
