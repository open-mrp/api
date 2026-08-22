package consumptionep

import (
	"context"
	"fmt"

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

type ConsumptionSvc interface {
	GetConsumption(ctx context.Context, req *RetrieveConsumptionRequest) (*apiresource.Consumption, *apierror.APIError)
	CreateConsumption(ctx context.Context, req *CreateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError)
	UpdateConsumption(ctx context.Context, req *UpdateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError)
	DeleteConsumption(ctx context.Context, req *DeleteConsumptionRequest) (*apiresource.Consumption, *apierror.APIError)
}

type ConsumptionSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type consumptionSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var consumptionSvcTracer = tracing.GetTracer("api-gateway.endpoints.consumptions.service")

func (c *ConsumptionSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("consumption endpoint service: core client is required")
	}
	return nil
}

func NewConsumptionSvc(config *ConsumptionSvcConfig) ConsumptionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &consumptionSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *consumptionSvcImpl) GetConsumption(ctx context.Context, req *RetrieveConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
	pbReq := &pb.GetConsumptionRequest{
		ProductionStepId: req.ProductionStepID,
		Id:               req.ConsumptionID,
		Includes:         resourcekit.FilterIncludes(ctx, consumptionIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, consumptionSvcTracer, "service.consumptions.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetConsumptionResponse, error) {
			return m.coreClient.GetConsumption(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := consumptionFromProto(resp.Consumption)
	stashConsumptionMeta(meta, resp.Consumption)
	return &result, nil
}

func (m *consumptionSvcImpl) CreateConsumption(ctx context.Context, req *CreateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
	pbReq := &pb.CreateConsumptionRequest{
		ProductionStepId:    req.ProductionStepID,
		ItemId:              req.ItemID,
		QuantityValue:       req.QuantityValue,
		QuantityUnitId:      req.QuantityUnitID,
		WasteQuantityValue:  req.WasteQuantityValue,
		WasteQuantityUnitId: req.WasteQuantityUnitID,
		Instructions:        req.Instructions.Ptr(),
		Includes:            resourcekit.FilterIncludes(ctx, consumptionIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, consumptionSvcTracer, "service.consumptions.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateConsumptionResponse, error) {
			return m.coreClient.CreateConsumption(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := consumptionFromProto(resp.Consumption)
	stashConsumptionMeta(meta, resp.Consumption)
	return &result, nil
}

func (m *consumptionSvcImpl) UpdateConsumption(ctx context.Context, req *UpdateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
	pbReq := &pb.UpdateConsumptionRequest{
		ProductionStepId:    req.ProductionStepID,
		Id:                  req.ConsumptionID,
		ItemId:              req.ItemID.Ptr(),
		QuantityValue:       req.QuantityValue.Ptr(),
		QuantityUnitId:      req.QuantityUnitID.Ptr(),
		WasteQuantityValue:  req.WasteQuantityValue.Ptr(),
		WasteQuantityUnitId: req.WasteQuantityUnitID.Ptr(),
		Instructions:        req.Instructions.Ptr(),
		Includes:            resourcekit.FilterIncludes(ctx, consumptionIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, consumptionSvcTracer, "service.consumptions.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateConsumptionResponse, error) {
			return m.coreClient.UpdateConsumption(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := consumptionFromProto(resp.Consumption)
	stashConsumptionMeta(meta, resp.Consumption)
	return &result, nil
}

func (m *consumptionSvcImpl) DeleteConsumption(ctx context.Context, req *DeleteConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
	pbReq := &pb.DeleteConsumptionRequest{
		ProductionStepId: req.ProductionStepID,
		Id:               req.ConsumptionID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, consumptionSvcTracer, "service.consumptions.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteConsumptionResponse, error) {
			return m.coreClient.DeleteConsumption(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := consumptionFromProto(resp.Consumption)
	stashConsumptionMeta(meta, resp.Consumption)
	return &result, nil
}

var consumptionIncludes = []string{"consumed_item"}

func consumptionQuantityFromProto(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	norm := apiresource.NormalizeQuantityValue(q.Value, q.UnitType)
	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType),
	}
}

func consumptionFromProto(c *pb.ConsumptionInfo) apiresource.Consumption {
	if c == nil {
		return apiresource.Consumption{}
	}

	return apiresource.Consumption{
		ID:            c.Id,
		Object:        constants.ObjectTypeConsumption,
		Quantity:      consumptionQuantityFromProto(c.Quantity),
		WasteQuantity: consumptionQuantityFromProto(c.WasteQuantity),
		Instructions:  c.Instructions,
		CreatedAt:     grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:     grpcutil.TimestampToTime(c.UpdatedAt),
	}
}

func stashConsumptionMeta(meta *resourcekit.LoadMeta, c *pb.ConsumptionInfo) {
	if c == nil {
		return
	}

	if c.ItemId != "" {
		itemType := constants.ItemTypeCode(c.ItemTypeCode)
		if !itemType.IsValid() {
			itemType = constants.ItemTypeCodeProduct
		}
		meta.Set(constants.ObjectTypeConsumption, c.Id, "consumed_item", &apiresource.Item{
			ID:           c.ItemId,
			Object:       constants.ObjectTypeItem,
			SKU:          c.ItemSku,
			Description:  c.ItemDescription,
			ItemTypeCode: itemType,
			CreatedAt:    grpcutil.TimestampToTime(c.CreatedAt),
			UpdatedAt:    grpcutil.TimestampToTime(c.UpdatedAt),
		})
	}
}
