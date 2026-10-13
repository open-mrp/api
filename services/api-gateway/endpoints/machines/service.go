package machineep

import (
	"context"
	"fmt"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MachineSvc interface {
	ListMachines(ctx context.Context, req *ListMachinesRequest) (*apiresource.List[apiresource.Machine], *apierror.APIError)
	ExportMachines(ctx context.Context, req *ExportMachinesRequest) (*apiresource.Job, *apierror.APIError)
	GetMachine(ctx context.Context, req *RetrieveMachineRequest) (*apiresource.Machine, *apierror.APIError)
	CreateMachine(ctx context.Context, req *CreateMachineRequest) (*apiresource.Machine, *apierror.APIError)
	UpdateMachine(ctx context.Context, req *UpdateMachineRequest) (*apiresource.Machine, *apierror.APIError)
	BulkUpsertMachines(ctx context.Context, req *BulkUpsertMachinesRequest) (*apiresource.Job, *apierror.APIError)
	DeleteMachine(ctx context.Context, req *DeleteMachineRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListMachineItems(ctx context.Context, req *ListMachineItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError)
	ReplaceMachineItems(ctx context.Context, req *ReplaceMachineItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError)
	SuggestMachineItemsFromScans(ctx context.Context, req *SuggestMachineItemsFromScansRequest) (*apiresource.List[apiresource.MachineItemScanSuggestion], *apierror.APIError)
}

type MachineSvcConfig struct {
	// CoreClient (required) is the core-service fulfillment gRPC client.
	CoreClient pb.CoreFulfillmentServiceClient
}

type machineSvcImpl struct {
	coreClient pb.CoreFulfillmentServiceClient
}

var machineSvcTracer = tracing.GetTracer("api-gateway.endpoints.machines.service")

func (c *MachineSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("machine endpoint service: core client is required")
	}
	return nil
}

func NewMachineSvc(config *MachineSvcConfig) MachineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &machineSvcImpl{
		coreClient: config.CoreClient,
	}
}

func loadMachineByID(ctx context.Context, id string) (*apiresource.Machine, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadMachines(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Machine not found.")
	}
	return v.(*apiresource.Machine), nil
}

func (m *machineSvcImpl) ExportMachines(ctx context.Context, req *ExportMachinesRequest) (*apiresource.Job, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, machineSvcTracer, "service.machines.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportMachinesResponse, error) {
			return m.coreClient.ExportMachines(ctx, &pb.ExportMachinesRequest{Query: req.Query}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *machineSvcImpl) ListMachines(ctx context.Context, req *ListMachinesRequest) (*apiresource.List[apiresource.Machine], *apierror.APIError) {
	pbReq := &pb.ListMachinesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, machineSvcTracer, "service.machines.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMachinesResponse, error) {
			return m.coreClient.ListMachines(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Machines))
	for i, machine := range resp.Machines {
		ids[i] = machine.Id
	}

	loaded, apiErr := resourceloaders.LoadMachines(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}

	machines := make([]apiresource.Machine, 0, len(resp.Machines))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			machines = append(machines, *v.(*apiresource.Machine))
		}
	}

	return apiresource.NewList(machines, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *machineSvcImpl) GetMachine(ctx context.Context, req *RetrieveMachineRequest) (*apiresource.Machine, *apierror.APIError) {
	return loadMachineByID(ctx, req.MachineID)
}

func (m *machineSvcImpl) CreateMachine(ctx context.Context, req *CreateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
	pbReq := &pb.CreateMachineRequest{
		Name:         req.Name,
		SerialNumber: req.SerialNumber,
		Notes:        req.Notes.Ptr(),
		DepartmentId: req.DepartmentID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, machineSvcTracer, "service.machines.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateMachineResponse, error) {
			return m.coreClient.CreateMachine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadMachineByID(ctx, resp.Machine.Id)
}

func (m *machineSvcImpl) BulkUpsertMachines(ctx context.Context, req *BulkUpsertMachinesRequest) (*apiresource.Job, *apierror.APIError) {
	pbMachines := make([]*pb.UpsertMachineInput, len(req.Machines))
	for i, machine := range req.Machines {
		pbMachines[i] = &pb.UpsertMachineInput{
			Name:         machine.Name,
			SerialNumber: machine.SerialNumber,
			Notes:        machine.Notes.Ptr(),
			Department:   apirequest.ObjectIdentifierToProto(machine.Department),
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, machineSvcTracer, "service.machines.bulk_upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkUpsertMachinesResponse, error) {
			return m.coreClient.BulkUpsertMachines(ctx, &pb.BulkUpsertMachinesRequest{Machines: pbMachines}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *machineSvcImpl) UpdateMachine(ctx context.Context, req *UpdateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
	pbReq := &pb.UpdateMachineRequest{
		Id:           req.MachineID,
		Name:         req.Name.Ptr(),
		SerialNumber: req.SerialNumber.Ptr(),
		Notes:        field.StringClearableToProto(req.Notes),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, machineSvcTracer, "service.machines.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateMachineResponse, error) {
			return m.coreClient.UpdateMachine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadMachineByID(ctx, resp.Machine.Id)
}

func (m *machineSvcImpl) DeleteMachine(ctx context.Context, req *DeleteMachineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteMachineRequest{
		Id: req.MachineID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, machineSvcTracer, "service.machines.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteMachine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func loadItemsByIDs(ctx context.Context, ids []string) ([]apiresource.Item, *apierror.APIError) {
	if len(ids) == 0 {
		return []apiresource.Item{}, nil
	}
	loaded, apiErr := resourceloaders.LoadItems(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.Item, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *v.(*apiresource.Item))
		}
	}
	return items, nil
}

func (m *machineSvcImpl) ListMachineItems(ctx context.Context, req *ListMachineItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, machineSvcTracer, "service.machines.list_items", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMachineItemsResponse, error) {
			return m.coreClient.ListMachineItems(ctx, &pb.ListMachineItemsRequest{
				MachineId: req.MachineID,
				Cursor:    req.Cursor,
				Limit:     req.Limit,
				Query:     req.Query,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Items))
	for i, item := range resp.Items {
		ids[i] = item.ItemId
	}
	items, apiErr := loadItemsByIDs(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *machineSvcImpl) ReplaceMachineItems(ctx context.Context, req *ReplaceMachineItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, machineSvcTracer, "service.machines.replace_items", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMachineItemsResponse, error) {
			return m.coreClient.ReplaceMachineItems(ctx, &pb.ReplaceMachineItemsRequest{
				MachineId: req.MachineID,
				ItemIds:   req.ItemIDs,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Items))
	for i, item := range resp.Items {
		ids[i] = item.ItemId
	}
	items, apiErr := loadItemsByIDs(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *machineSvcImpl) SuggestMachineItemsFromScans(ctx context.Context, req *SuggestMachineItemsFromScansRequest) (*apiresource.List[apiresource.MachineItemScanSuggestion], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, machineSvcTracer, "service.machines.suggest_items_from_scans", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SuggestMachineItemsFromScansResponse, error) {
			return m.coreClient.SuggestMachineItemsFromScans(ctx, &pb.SuggestMachineItemsFromScansRequest{
				MachineId:    req.MachineID,
				WindowMonths: req.WindowMonths.Ptr(),
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	suggestions := make([]apiresource.MachineItemScanSuggestion, 0, len(resp.Suggestions))
	for _, s := range resp.Suggestions {
		row := apiresource.MachineItemScanSuggestion{
			ID:            s.ItemId,
			Object:        constants.ObjectTypeMachineItemScanSuggestion,
			LastScannedAt: s.LastScannedAt.AsTime(),
			BatchCount:    s.BatchCount,
		}
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeMachineItemScanSuggestion, s.ItemId, "item_id", s.ItemId)
		suggestions = append(suggestions, row)
	}
	return apiresource.NewList(suggestions, apiresource.PageInfo{}), nil
}
