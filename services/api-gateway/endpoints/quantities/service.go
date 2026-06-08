package quantityep

import (
	"context"
	"fmt"

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

type QuantitySvc interface {
	UpdateQuantity(ctx context.Context, req *UpdateQuantityRequest) (*apiresource.Quantity, *apierror.APIError)
}

type QuantitySvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type quantitySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var quantitySvcTracer = tracing.GetTracer("api-gateway.endpoints.quantities.service")

func (c *QuantitySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("quantity endpoint service: core client is required")
	}
	return nil
}

func NewQuantitySvc(config *QuantitySvcConfig) QuantitySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &quantitySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *quantitySvcImpl) UpdateQuantity(ctx context.Context, req *UpdateQuantityRequest) (*apiresource.Quantity, *apierror.APIError) {
	pbReq := &pb.UpdateQuantityRequest{
		Id:         req.QuantityID,
		Value:      req.Value.Ptr(),
		UnitId:     req.UnitID.Ptr(),
		ObjectId:   req.ObjectID.Ptr(),
		ObjectType: req.ObjectType.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, quantitySvcTracer, "service.quantities.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateQuantityResponse, error) {
			return m.coreClient.UpdateQuantity(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := quantityFromProto(resp.Quantity)
	stashQuantityMeta(meta, resp.Quantity)
	return &result, nil
}

func quantityFromProto(q *pb.QuantityInfo) apiresource.Quantity {
	if q == nil {
		return apiresource.Quantity{}
	}

	norm := apiresource.NormalizeQuantityValue(q.Value, q.UnitType)
	return apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType),
	}
}

func stashQuantityMeta(meta *resourcekit.LoadMeta, q *pb.QuantityInfo) {
	if q == nil {
		return
	}
	// If the proto carried full unit detail, stash the resolved Unit directly
	// (no extra fetch). Otherwise stash just the FK id so LoadUnits fetches the
	// real Unit on ?include=unit. Never fabricate.
	if unit := UnitFromQuantityInfo(q); unit != nil {
		meta.Set(constants.ObjectTypeQuantity, q.Id, "unit", unit)
	} else if q.UnitId != "" {
		meta.Set(constants.ObjectTypeQuantity, q.Id, "unit_id", q.UnitId)
	}
}
