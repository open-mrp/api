package manufacturingmethodep

import (
	"context"
	"fmt"
	"strconv"

	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ManufacturingMethodSvc interface {
	ListManufacturingMethods(ctx context.Context, req *ListManufacturingMethodsRequest) (*apiresource.List[apiresource.ManufacturingMethod], *apierror.APIError)
	GetManufacturingMethod(ctx context.Context, req *RetrieveManufacturingMethodRequest) (*apiresource.ManufacturingMethod, *apierror.APIError)
	CreateManufacturingMethod(ctx context.Context, req *CreateManufacturingMethodRequest) (*apiresource.ManufacturingMethod, *apierror.APIError)
	UpdateManufacturingMethod(ctx context.Context, req *UpdateManufacturingMethodRequest) (*apiresource.ManufacturingMethod, *apierror.APIError)
	DeleteManufacturingMethod(ctx context.Context, req *DeleteManufacturingMethodRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ActivateManufacturingMethod(ctx context.Context, req *lifecycle.Request) (*apiresource.ManufacturingMethod, *apierror.APIError)
	NewManufacturingMethodVersion(ctx context.Context, req *lifecycle.Request) (*apiresource.ManufacturingMethod, *apierror.APIError)
	GetItemProductionFlow(ctx context.Context, req *GetProductionFlowRequest) (*apiresource.List[apiresource.ManufacturingMethod], *apierror.APIError)

	ListMethodOperations(ctx context.Context, req *ListMethodOperationsRequest) (*apiresource.List[apiresource.MethodOperation], *apierror.APIError)
	GetMethodOperation(ctx context.Context, req *RetrieveMethodOperationRequest) (*apiresource.MethodOperation, *apierror.APIError)
	CreateMethodOperation(ctx context.Context, req *CreateMethodOperationRequest) (*apiresource.MethodOperation, *apierror.APIError)
	UpdateMethodOperation(ctx context.Context, req *UpdateMethodOperationRequest) (*apiresource.MethodOperation, *apierror.APIError)
	DeleteMethodOperation(ctx context.Context, req *DeleteMethodOperationRequest) (*apiresource.EmptyResource, *apierror.APIError)

	ListMethodMaterials(ctx context.Context, req *ListMethodMaterialsRequest) (*apiresource.List[apiresource.MethodMaterial], *apierror.APIError)
	GetMethodMaterial(ctx context.Context, req *RetrieveMethodMaterialRequest) (*apiresource.MethodMaterial, *apierror.APIError)
	CreateMethodMaterial(ctx context.Context, req *CreateMethodMaterialRequest) (*apiresource.MethodMaterial, *apierror.APIError)
	UpdateMethodMaterial(ctx context.Context, req *UpdateMethodMaterialRequest) (*apiresource.MethodMaterial, *apierror.APIError)
	DeleteMethodMaterial(ctx context.Context, req *DeleteMethodMaterialRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ManufacturingMethodSvcConfig struct {
	// CoreClient (required) is the core-service manufacturing method gRPC client.
	CoreClient pb.CoreManufacturingMethodServiceClient
}

func (c *ManufacturingMethodSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("manufacturing method endpoint service: core client is required")
	}
	return nil
}

type manufacturingMethodSvcImpl struct {
	coreClient pb.CoreManufacturingMethodServiceClient
}

var manufacturingMethodSvcTracer = tracing.GetTracer("api-gateway.endpoints.manufacturing_methods.service")

func NewManufacturingMethodSvc(config *ManufacturingMethodSvcConfig) ManufacturingMethodSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &manufacturingMethodSvcImpl{coreClient: config.CoreClient}
}

func quantityInputToProto(q apirequest.QuantityInput) *pb.MethodQuantityInput {
	return &pb.MethodQuantityInput{Value: q.Value, UnitId: q.UnitID}
}

func optionalQuantityToProto(f field.Optional[apirequest.QuantityInput]) *pb.MethodQuantityInput {
	if v, ok := f.Value(); ok {
		return quantityInputToProto(v)
	}
	return nil
}

func methodsFromProto(ctx context.Context, infos []*pb.ManufacturingMethodInfo) []apiresource.ManufacturingMethod {
	out := make([]apiresource.ManufacturingMethod, len(infos))
	for i, m := range infos {
		out[i] = *resourceloaders.ManufacturingMethodFromProto(ctx, m)
	}
	return out
}

func (s *manufacturingMethodSvcImpl) ListManufacturingMethods(ctx context.Context, req *ListManufacturingMethodsRequest) (*apiresource.List[apiresource.ManufacturingMethod], *apierror.APIError) {
	if apiErr := req.CreatedRange.Validate(); apiErr != nil {
		return nil, apiErr
	}
	statuses := make([]string, len(req.Statuses))
	for i, st := range req.Statuses {
		statuses[i] = string(st)
	}
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListManufacturingMethodsResponse, error) {
			return s.coreClient.ListManufacturingMethods(ctx, &pb.ListManufacturingMethodsRequest{
				Cursor: req.Cursor, Limit: req.Limit, ItemIds: req.ItemIDs, Statuses: statuses,
				WorkCenterIds: req.WorkCenterIDs, Created: req.CreatedRange.ToProto(),
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return apiresource.NewList(methodsFromProto(ctx, resp.ManufacturingMethods), grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (s *manufacturingMethodSvcImpl) GetManufacturingMethod(ctx context.Context, req *RetrieveManufacturingMethodRequest) (*apiresource.ManufacturingMethod, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetManufacturingMethodsByIDsResponse, error) {
			return s.coreClient.BatchGetManufacturingMethodsByIDs(ctx, &pb.BatchGetManufacturingMethodsByIDsRequest{Ids: []string{req.ID}}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(resp.ManufacturingMethods) == 0 {
		return nil, apierror.NewNotFoundError("Manufacturing method not found.")
	}
	return resourceloaders.ManufacturingMethodFromProto(ctx, resp.ManufacturingMethods[0]), nil
}

func (s *manufacturingMethodSvcImpl) methodCall(ctx context.Context, span string, call func(ctx context.Context, opts ...grpc.CallOption) (*pb.ManufacturingMethodResponse, error)) (*apiresource.ManufacturingMethod, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, span, domain.ServiceName, call)
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.ManufacturingMethodFromProto(ctx, resp.ManufacturingMethod), nil
}

func (s *manufacturingMethodSvcImpl) CreateManufacturingMethod(ctx context.Context, req *CreateManufacturingMethodRequest) (*apiresource.ManufacturingMethod, *apierror.APIError) {
	return s.methodCall(ctx, "service.manufacturing_methods.create", func(ctx context.Context, opts ...grpc.CallOption) (*pb.ManufacturingMethodResponse, error) {
		return s.coreClient.CreateManufacturingMethod(ctx, &pb.CreateManufacturingMethodRequest{
			ItemId: req.ItemID, OutputQuantity: optionalQuantityToProto(req.OutputQuantity),
		}, opts...)
	})
}

func (s *manufacturingMethodSvcImpl) UpdateManufacturingMethod(ctx context.Context, req *UpdateManufacturingMethodRequest) (*apiresource.ManufacturingMethod, *apierror.APIError) {
	return s.methodCall(ctx, "service.manufacturing_methods.update", func(ctx context.Context, opts ...grpc.CallOption) (*pb.ManufacturingMethodResponse, error) {
		return s.coreClient.UpdateManufacturingMethod(ctx, &pb.UpdateManufacturingMethodRequest{
			Id: req.ID, OutputQuantity: optionalQuantityToProto(req.OutputQuantity),
		}, opts...)
	})
}

func (s *manufacturingMethodSvcImpl) DeleteManufacturingMethod(ctx context.Context, req *DeleteManufacturingMethodRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return s.coreClient.DeleteManufacturingMethod(ctx, &pb.ManufacturingMethodIDRequest{Id: req.ID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *manufacturingMethodSvcImpl) ActivateManufacturingMethod(ctx context.Context, req *lifecycle.Request) (*apiresource.ManufacturingMethod, *apierror.APIError) {
	return s.methodCall(ctx, "service.manufacturing_methods.activate", func(ctx context.Context, opts ...grpc.CallOption) (*pb.ManufacturingMethodResponse, error) {
		return s.coreClient.ActivateManufacturingMethod(ctx, &pb.ManufacturingMethodIDRequest{Id: req.ID}, opts...)
	})
}

func (s *manufacturingMethodSvcImpl) NewManufacturingMethodVersion(ctx context.Context, req *lifecycle.Request) (*apiresource.ManufacturingMethod, *apierror.APIError) {
	return s.methodCall(ctx, "service.manufacturing_methods.new_version", func(ctx context.Context, opts ...grpc.CallOption) (*pb.ManufacturingMethodResponse, error) {
		return s.coreClient.NewManufacturingMethodVersion(ctx, &pb.ManufacturingMethodIDRequest{Id: req.ID}, opts...)
	})
}

func (s *manufacturingMethodSvcImpl) GetItemProductionFlow(ctx context.Context, req *GetProductionFlowRequest) (*apiresource.List[apiresource.ManufacturingMethod], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.production_flow", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetMethodProductionFlowResponse, error) {
			return s.coreClient.GetMethodProductionFlow(ctx, &pb.GetMethodProductionFlowRequest{ItemId: req.ItemID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	// The flow comes whole and in order; a cursor is an offset into it.
	ids := resp.ManufacturingMethodIds
	start := 0
	if req.Cursor != nil {
		cur, err := pagination.DecodeValueCursor(*req.Cursor)
		offset, convErr := strconv.Atoi(cur.Value)
		if err != nil || convErr != nil || offset < 0 || offset > len(ids) {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		start = offset
	}
	end := min(start+int(req.Limit), len(ids))
	page := &pb.PageInfo{}
	if end < len(ids) {
		page.NextCursor = new(pagination.EncodeValueCursor(pagination.ValueCursor{Value: strconv.Itoa(end), ID: ids[end], Direction: pagination.DirectionForward}))
	}
	if start > 0 {
		prev := max(start-int(req.Limit), 0)
		page.PrevCursor = new(pagination.EncodeValueCursor(pagination.ValueCursor{Value: strconv.Itoa(prev), ID: ids[prev], Direction: pagination.DirectionBackward}))
	}
	pageIDs := ids[start:end]
	loaded, apiErr := resourceloaders.LoadManufacturingMethods(ctx, pageIDs)
	if apiErr != nil {
		return nil, apiErr
	}
	methods := make([]apiresource.ManufacturingMethod, 0, len(pageIDs))
	for _, id := range pageIDs {
		if v, ok := loaded[id]; ok {
			methods = append(methods, *(v.(*apiresource.ManufacturingMethod)))
		}
	}
	return apiresource.NewList(methods, grpcutil.MapProtoPageInfo(ctx, page)), nil
}

// --- operations ---

func operationInputToProto(
	position field.Optional[int32], sequence field.Optional[constants.OperationSequence], workCenterID *string,
	machineIDs *[]string, station field.Clearable[string], setup field.Clearable[apirequest.QuantityInput],
	laborTime *apirequest.RateInput, machineTime field.Clearable[apirequest.RateInput],
	laborRate, overheadRate field.Optional[apirequest.UnitPriceInput], leveling, allowance field.Optional[string],
	instructions field.Clearable[string],
) *pb.MethodOperationInput {
	in := &pb.MethodOperationInput{
		Position:          position.Ptr(),
		WorkCenterId:      workCenterID,
		ScanningStationId: fieldpb.StringClearableToProto(station),
		SetupTime:         apirequest.QuantityFieldToProto(setup),
		LaborRate:         apirequest.OptionalUnitPriceToProto(laborRate),
		OverheadRate:      apirequest.OptionalUnitPriceToProto(overheadRate),
		LevelingPercent:   leveling.Ptr(),
		AllowancePercent:  allowance.Ptr(),
		WorkInstructions:  fieldpb.StringClearableToProto(instructions),
	}
	if v, ok := sequence.Value(); ok {
		str := string(v)
		in.Sequence = &str
	}
	if machineIDs != nil {
		in.MachineIds = &pb.StringListPatch{Clear: len(*machineIDs) == 0, Value: *machineIDs}
	}
	if laborTime != nil {
		in.LaborTime = laborTime.ToProto()
	}
	switch {
	case machineTime.IsClear():
		in.MachineTime = &pb.MethodRatePatch{Clear: true}
	case !machineTime.IsUnset():
		v, _ := machineTime.Value()
		in.MachineTime = &pb.MethodRatePatch{Value: v.ToProto()}
	}
	return in
}

func (s *manufacturingMethodSvcImpl) ListMethodOperations(ctx context.Context, req *ListMethodOperationsRequest) (*apiresource.List[apiresource.MethodOperation], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.list_operations", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMethodOperationsResponse, error) {
			return s.coreClient.ListMethodOperations(ctx, &pb.ListMethodMembersRequest{
				ManufacturingMethodId: req.ManufacturingMethodID, Cursor: req.Cursor, Limit: req.Limit,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	ops := make([]apiresource.MethodOperation, len(resp.Operations))
	for i, op := range resp.Operations {
		ops[i] = *resourceloaders.MethodOperationFromProto(ctx, op)
	}
	return apiresource.NewList(ops, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (s *manufacturingMethodSvcImpl) GetMethodOperation(ctx context.Context, req *RetrieveMethodOperationRequest) (*apiresource.MethodOperation, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.get_operation", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetMethodOperationsByIDsResponse, error) {
			return s.coreClient.BatchGetMethodOperationsByIDs(ctx, &pb.BatchGetMethodOperationsByIDsRequest{Ids: []string{req.ID}}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(resp.Operations) == 0 || resp.Operations[0].ManufacturingMethodId != req.ManufacturingMethodID {
		return nil, apierror.NewNotFoundError("Operation not found.")
	}
	return resourceloaders.MethodOperationFromProto(ctx, resp.Operations[0]), nil
}

func (s *manufacturingMethodSvcImpl) operationCall(ctx context.Context, span string, call func(ctx context.Context, opts ...grpc.CallOption) (*pb.MethodOperationResponse, error)) (*apiresource.MethodOperation, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, span, domain.ServiceName, call)
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.MethodOperationFromProto(ctx, resp.Operation), nil
}

func (s *manufacturingMethodSvcImpl) CreateMethodOperation(ctx context.Context, req *CreateMethodOperationRequest) (*apiresource.MethodOperation, *apierror.APIError) {
	var machines *[]string
	if req.MachineIDs != nil {
		machines = &req.MachineIDs
	}
	in := operationInputToProto(req.Position, req.Sequence, &req.WorkCenterID, machines,
		optionalToClearable(req.ScanningStationID), optionalToClearable(req.SetupTime), &req.LaborTime,
		optionalToClearable(req.MachineTime), req.LaborRate, req.OverheadRate, req.LevelingPercent, req.AllowancePercent,
		optionalToClearable(req.WorkInstructions))
	return s.operationCall(ctx, "service.manufacturing_methods.create_operation", func(ctx context.Context, opts ...grpc.CallOption) (*pb.MethodOperationResponse, error) {
		return s.coreClient.CreateMethodOperation(ctx, &pb.CreateMethodOperationRequest{ManufacturingMethodId: req.ManufacturingMethodID, Operation: in}, opts...)
	})
}

// optionalToClearable carries a create field into the shared write shape, where unset stays unset.
func optionalToClearable[T any](f field.Optional[T]) field.Clearable[T] {
	if v, ok := f.Value(); ok {
		return field.Set(v)
	}
	return field.Clearable[T]{}
}

func (s *manufacturingMethodSvcImpl) UpdateMethodOperation(ctx context.Context, req *UpdateMethodOperationRequest) (*apiresource.MethodOperation, *apierror.APIError) {
	var laborTime *apirequest.RateInput
	if v, ok := req.LaborTime.Value(); ok {
		laborTime = &v
	}
	var machines *[]string
	if v, ok := req.MachineIDs.Value(); ok {
		machines = &v
	}
	in := operationInputToProto(req.Position, req.Sequence, req.WorkCenterID.Ptr(), machines, req.ScanningStationID, req.SetupTime,
		laborTime, req.MachineTime, req.LaborRate, req.OverheadRate, req.LevelingPercent, req.AllowancePercent, req.WorkInstructions)
	return s.operationCall(ctx, "service.manufacturing_methods.update_operation", func(ctx context.Context, opts ...grpc.CallOption) (*pb.MethodOperationResponse, error) {
		return s.coreClient.UpdateMethodOperation(ctx, &pb.UpdateMethodOperationRequest{ManufacturingMethodId: req.ManufacturingMethodID, Id: req.ID, Operation: in}, opts...)
	})
}

func (s *manufacturingMethodSvcImpl) DeleteMethodOperation(ctx context.Context, req *DeleteMethodOperationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.delete_operation", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return s.coreClient.DeleteMethodOperation(ctx, &pb.MethodMemberIDRequest{ManufacturingMethodId: req.ManufacturingMethodID, Id: req.ID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// --- materials ---

func (s *manufacturingMethodSvcImpl) ListMethodMaterials(ctx context.Context, req *ListMethodMaterialsRequest) (*apiresource.List[apiresource.MethodMaterial], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.list_materials", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMethodMaterialsResponse, error) {
			return s.coreClient.ListMethodMaterials(ctx, &pb.ListMethodMembersRequest{
				ManufacturingMethodId: req.ManufacturingMethodID, Cursor: req.Cursor, Limit: req.Limit,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	materials := make([]apiresource.MethodMaterial, len(resp.Materials))
	for i, m := range resp.Materials {
		materials[i] = *resourceloaders.MethodMaterialFromProto(ctx, m)
	}
	return apiresource.NewList(materials, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (s *manufacturingMethodSvcImpl) GetMethodMaterial(ctx context.Context, req *RetrieveMethodMaterialRequest) (*apiresource.MethodMaterial, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.get_material", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetMethodMaterialsByIDsResponse, error) {
			return s.coreClient.BatchGetMethodMaterialsByIDs(ctx, &pb.BatchGetMethodMaterialsByIDsRequest{Ids: []string{req.ID}}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(resp.Materials) == 0 || resp.Materials[0].ManufacturingMethodId != req.ManufacturingMethodID {
		return nil, apierror.NewNotFoundError("Material not found.")
	}
	return resourceloaders.MethodMaterialFromProto(ctx, resp.Materials[0]), nil
}

func (s *manufacturingMethodSvcImpl) materialCall(ctx context.Context, span string, call func(ctx context.Context, opts ...grpc.CallOption) (*pb.MethodMaterialResponse, error)) (*apiresource.MethodMaterial, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, span, domain.ServiceName, call)
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.MethodMaterialFromProto(ctx, resp.Material), nil
}

func supplyPtr(f field.Optional[constants.Supply]) *string {
	if v, ok := f.Value(); ok {
		s := string(v)
		return &s
	}
	return nil
}

func (s *manufacturingMethodSvcImpl) CreateMethodMaterial(ctx context.Context, req *CreateMethodMaterialRequest) (*apiresource.MethodMaterial, *apierror.APIError) {
	in := &pb.MethodMaterialInput{
		ItemId:           &req.ItemID,
		OperationId:      req.OperationID.Ptr(),
		Quantity:         quantityInputToProto(req.Quantity),
		ScrapQuantity:    optionalQuantityToProto(req.ScrapQuantity),
		Supply:           supplyPtr(req.Supply),
		WorkInstructions: fieldpb.StringClearableToProto(optionalToClearable(req.WorkInstructions)),
	}
	return s.materialCall(ctx, "service.manufacturing_methods.create_material", func(ctx context.Context, opts ...grpc.CallOption) (*pb.MethodMaterialResponse, error) {
		return s.coreClient.CreateMethodMaterial(ctx, &pb.CreateMethodMaterialRequest{ManufacturingMethodId: req.ManufacturingMethodID, Material: in}, opts...)
	})
}

func (s *manufacturingMethodSvcImpl) UpdateMethodMaterial(ctx context.Context, req *UpdateMethodMaterialRequest) (*apiresource.MethodMaterial, *apierror.APIError) {
	in := &pb.MethodMaterialInput{
		ItemId:           req.ItemID.Ptr(),
		OperationId:      req.OperationID.Ptr(),
		Quantity:         optionalQuantityToProto(req.Quantity),
		ScrapQuantity:    optionalQuantityToProto(req.ScrapQuantity),
		Supply:           supplyPtr(req.Supply),
		WorkInstructions: fieldpb.StringClearableToProto(req.WorkInstructions),
	}
	return s.materialCall(ctx, "service.manufacturing_methods.update_material", func(ctx context.Context, opts ...grpc.CallOption) (*pb.MethodMaterialResponse, error) {
		return s.coreClient.UpdateMethodMaterial(ctx, &pb.UpdateMethodMaterialRequest{ManufacturingMethodId: req.ManufacturingMethodID, Id: req.ID, Material: in}, opts...)
	})
}

func (s *manufacturingMethodSvcImpl) DeleteMethodMaterial(ctx context.Context, req *DeleteMethodMaterialRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, manufacturingMethodSvcTracer, "service.manufacturing_methods.delete_material", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return s.coreClient.DeleteMethodMaterial(ctx, &pb.MethodMemberIDRequest{ManufacturingMethodId: req.ManufacturingMethodID, Id: req.ID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}
