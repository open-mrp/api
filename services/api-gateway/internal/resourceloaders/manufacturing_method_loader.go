package resourceloaders

import (
	"context"
	"strings"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var manufacturingMethodLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.manufacturing_method")

// manufacturingMethodClient is the CoreManufacturingMethodService client. Set at startup.
var manufacturingMethodClient pb.CoreManufacturingMethodServiceClient

// SetManufacturingMethodClient is called once at startup with the CoreManufacturingMethodService client.
func SetManufacturingMethodClient(c pb.CoreManufacturingMethodServiceClient) {
	manufacturingMethodClient = c
}

// Load meta keys a method carries for its embedded lists.
const (
	MethodOperationIDsKey        = "operation_ids"
	MethodOperationsCursorKey    = "operations_next_cursor"
	MethodMaterialIDsKey         = "material_ids"
	MethodMaterialsCursorKey     = "materials_next_cursor"
	defaultMethodOfItemKeyPrefix = "default_manufacturing_method_of:"
)

// DefaultMethodOfItemKey is the key LoadManufacturingMethods resolves to the method an item is made by when nothing
// more specific applies, so an item's default_manufacturing_method loads in the same batch as any other method.
func DefaultMethodOfItemKey(itemID string) string {
	return defaultMethodOfItemKeyPrefix + itemID
}

// ManufacturingMethodMembersPath is the operations or materials list of a method, where its embedded list continues.
// members is "operations" or "materials".
func ManufacturingMethodMembersPath(methodID, members string) string {
	return "/v1/operations/manufacturing-methods/" + methodID + "/" + members
}

// LoadManufacturingMethods batch-loads methods by ID. A DefaultMethodOfItemKey loads the item's default method under
// that key; an item with none has no entry.
func LoadManufacturingMethods(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	var methodIDs, itemIDs []string
	for _, id := range ids {
		if itemID, ok := strings.CutPrefix(id, defaultMethodOfItemKeyPrefix); ok {
			itemIDs = append(itemIDs, itemID)
		} else {
			methodIDs = append(methodIDs, id)
		}
	}
	var defaults map[string]string
	if len(itemIDs) > 0 {
		resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodLoaderTracer, "loader.manufacturing_methods.defaults", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetDefaultManufacturingMethodIDsResponse, error) {
				return manufacturingMethodClient.GetDefaultManufacturingMethodIDs(ctx, &pb.GetDefaultManufacturingMethodIDsRequest{ItemIds: itemIDs}, opts...)
			})
		if apiErr != nil {
			return nil, apiErr
		}
		defaults = resp.GetMethodIds()
		for _, mid := range defaults {
			methodIDs = append(methodIDs, mid)
		}
	}
	out := make(map[string]any, len(ids))
	if len(methodIDs) == 0 {
		return out, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodLoaderTracer, "loader.manufacturing_methods.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetManufacturingMethodsByIDsResponse, error) {
			return manufacturingMethodClient.BatchGetManufacturingMethodsByIDs(ctx, &pb.BatchGetManufacturingMethodsByIDsRequest{Ids: methodIDs}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	for _, m := range resp.ManufacturingMethods {
		out[m.Id] = ManufacturingMethodFromProto(ctx, m)
	}
	for itemID, mid := range defaults {
		if v, ok := out[mid]; ok {
			out[DefaultMethodOfItemKey(itemID)] = v
		}
	}
	return out, nil
}

// ManufacturingMethodFromProto presents a method and stashes its item and first members in load meta for includes.
func ManufacturingMethodFromProto(ctx context.Context, m *pb.ManufacturingMethodInfo) *apiresource.ManufacturingMethod {
	meta := include.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeManufacturingMethod, m.Id, "item_id", m.ItemId)
	meta.Set(constants.ObjectTypeManufacturingMethod, m.Id, MethodOperationIDsKey, m.OperationIds)
	meta.Set(constants.ObjectTypeManufacturingMethod, m.Id, MethodOperationsCursorKey, m.GetOperationsNextCursor())
	meta.Set(constants.ObjectTypeManufacturingMethod, m.Id, MethodMaterialIDsKey, m.MaterialIds)
	meta.Set(constants.ObjectTypeManufacturingMethod, m.Id, MethodMaterialsCursorKey, m.GetMaterialsNextCursor())
	return &apiresource.ManufacturingMethod{
		ID:             m.Id,
		Object:         constants.ObjectTypeManufacturingMethod,
		Version:        m.Version,
		Status:         constants.ManufacturingMethodStatus(m.Status),
		OutputQuantity: quantityValueFromProto(m.OutputQuantity),
		CreatedAt:      grpcutil.TimestampToTime(m.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(m.UpdatedAt),
	}
}

func rateInputFromProto(r *pb.RateInput) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return apiresource.NewRate(apiresource.NormalizeRateValue(r.Value), r.UnitId, r.PerUnitId)
}

// LoadMethodOperations batch-loads operations by ID.
func LoadMethodOperations(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodLoaderTracer, "loader.method_operations.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetMethodOperationsByIDsResponse, error) {
			return manufacturingMethodClient.BatchGetMethodOperationsByIDs(ctx, &pb.BatchGetMethodOperationsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Operations))
	for _, op := range resp.Operations {
		out[op.Id] = MethodOperationFromProto(ctx, op)
	}
	return out, nil
}

// MethodOperationFromProto presents an operation and stashes its work center, machines and station for includes.
func MethodOperationFromProto(ctx context.Context, op *pb.MethodOperationInfo) *apiresource.MethodOperation {
	meta := include.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeMethodOperation, op.Id, "manufacturing_method_id", op.ManufacturingMethodId)
	meta.Set(constants.ObjectTypeMethodOperation, op.Id, "work_center_id", op.GetWorkCenterId())
	meta.Set(constants.ObjectTypeMethodOperation, op.Id, "machine_ids", op.MachineIds)
	meta.Set(constants.ObjectTypeMethodOperation, op.Id, "scanning_station_id", op.GetScanningStationId())
	return &apiresource.MethodOperation{
		ID:               op.Id,
		Object:           constants.ObjectTypeMethodOperation,
		Position:         op.Position,
		Sequence:         constants.OperationSequence(op.Sequence),
		Type:             constants.OperationType(op.Type),
		SetupTime:        quantityValueFromProto(op.SetupTime),
		LaborTime:        rateInputFromProto(op.LaborTime),
		MachineTime:      rateInputFromProto(op.MachineTime),
		LaborRate:        UnitPriceFromProto(op.LaborRate),
		OverheadRate:     UnitPriceFromProto(op.OverheadRate),
		LevelingPercent:  op.LevelingPercent,
		AllowancePercent: op.AllowancePercent,
		WorkInstructions: op.WorkInstructions,
		CreatedAt:        grpcutil.TimestampToTime(op.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(op.UpdatedAt),
	}
}

// LoadMethodMaterials batch-loads materials by ID.
func LoadMethodMaterials(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodLoaderTracer, "loader.method_materials.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetMethodMaterialsByIDsResponse, error) {
			return manufacturingMethodClient.BatchGetMethodMaterialsByIDs(ctx, &pb.BatchGetMethodMaterialsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Materials))
	for _, m := range resp.Materials {
		out[m.Id] = MethodMaterialFromProto(ctx, m)
	}
	return out, nil
}

// MethodMaterialFromProto presents a material and stashes its item and operation for includes.
func MethodMaterialFromProto(ctx context.Context, m *pb.MethodMaterialInfo) *apiresource.MethodMaterial {
	meta := include.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeMethodMaterial, m.Id, "manufacturing_method_id", m.ManufacturingMethodId)
	meta.Set(constants.ObjectTypeMethodMaterial, m.Id, "item_id", m.ItemId)
	meta.Set(constants.ObjectTypeMethodMaterial, m.Id, "operation_id", m.GetOperationId())
	return &apiresource.MethodMaterial{
		ID:               m.Id,
		Object:           constants.ObjectTypeMethodMaterial,
		Quantity:         quantityValueFromProto(m.Quantity),
		ScrapQuantity:    quantityValueFromProto(m.ScrapQuantity),
		Supply:           constants.Supply(m.Supply),
		WorkInstructions: m.WorkInstructions,
		CreatedAt:        grpcutil.TimestampToTime(m.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(m.UpdatedAt),
	}
}
