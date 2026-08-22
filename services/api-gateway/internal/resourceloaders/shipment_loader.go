package resourceloaders

import (
	"context"
	"time"

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
		// Stash the record-reference metadata (tracking number/URL, carrier, status, ship
		// date) so lightweight Record references to this shipment — e.g. a sales order's
		// related.shipments — can carry a shipment preview without expanding the full resource.
		if meta := shipmentRecordMetadata(resp.Shipment); len(meta) > 0 {
			resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeShipment, resp.Shipment.Id, "record_metadata", meta)
		}
	}
	return out, nil
}

// shipmentRecordMetadata builds the type-specific metadata surfaced on a shipment Record reference: the master tracking number, a carrier tracking deep-link, carrier name/code, status, and ship date. Keys are omitted when their source value is empty.
func shipmentRecordMetadata(s *pb.ShipmentInfo) map[string]string {
	meta := map[string]string{}
	tracking := ""
	if s.MasterTrackingNumber != nil {
		tracking = *s.MasterTrackingNumber
	}
	if tracking != "" {
		meta["tracking_number"] = tracking
	}
	if s.CarrierName != "" {
		meta["carrier"] = s.CarrierName
	}
	if s.CarrierCode != nil && *s.CarrierCode != "" {
		meta["carrier_code"] = *s.CarrierCode
		if url := constants.TrackingURL(constants.CarrierCode(*s.CarrierCode), tracking); url != "" {
			meta["tracking_url"] = url
		}
	}
	if s.StatusCode != "" {
		meta["status"] = s.StatusCode
	}
	if s.ShippedAt != nil {
		meta["shipped_at"] = s.ShippedAt.AsTime().Format(time.RFC3339)
	}
	return meta
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
		Priority:             constants.PriorityCode(s.PriorityCode),
		CaseCount:            int32(s.CaseCount),
		IsReadyToShip:        s.IsReadyToShip,
		CreatedAt:            grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

// Satisfies Register's required Load but is never invoked: shipment lines are expandable only by
// traversal from their shipment (attached inline via ExtractRefs), never loaded by their own id.
func LoadShipmentLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadShipmentLines must not be called — shipment lines are attached by the shipment and traversed via ExtractRefs, not loaded by id",
	)
}
