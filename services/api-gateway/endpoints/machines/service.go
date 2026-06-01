package machineep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MachineSvc interface {
	ListMachines(ctx context.Context, req *ListMachinesRequest) (*apiresource.List[apiresource.Machine], *apierror.APIError)
	GetMachine(ctx context.Context, req *RetrieveMachineRequest) (*apiresource.Machine, *apierror.APIError)
	CreateMachine(ctx context.Context, req *CreateMachineRequest) (*apiresource.Machine, *apierror.APIError)
	UpdateMachine(ctx context.Context, req *UpdateMachineRequest) (*apiresource.Machine, *apierror.APIError)
	DeleteMachine(ctx context.Context, req *DeleteMachineRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type MachineSvcConfig struct {
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
		Notes:        req.Notes,
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

func (m *machineSvcImpl) UpdateMachine(ctx context.Context, req *UpdateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
	pbReq := &pb.UpdateMachineRequest{
		Id:           req.MachineID,
		Name:         req.Name,
		SerialNumber: req.SerialNumber,
		Notes:        req.Notes,
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
