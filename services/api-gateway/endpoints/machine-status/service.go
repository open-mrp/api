package machinestatusep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var machineStatusSvcTracer = tracing.GetTracer("api-gateway.machine_status_service")

type MachineStatusSvc interface {
	ListMachineStatus(ctx context.Context, req *ListMachineStatusRequest) (*apiresource.List[apiresource.MachineStatus], *apierror.APIError)
}

type MachineStatusSvcConfig struct {
	// CoreClient (required) is the core-service fulfillment gRPC client.
	CoreClient pb.CoreFulfillmentServiceClient
}

func (c *MachineStatusSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("machine status endpoint service: core client is required")
	}
	return nil
}

type machineStatusSvcImpl struct {
	coreClient pb.CoreFulfillmentServiceClient
}

func NewMachineStatusSvc(cfg *MachineStatusSvcConfig) MachineStatusSvc {
	if err := cfg.validate(); err != nil {
		panic(err)
	}

	return &machineStatusSvcImpl{coreClient: cfg.CoreClient}
}

func campaignFromProto(info *pb.MachineCampaignInfo) *apiresource.MachineCampaign {
	if info == nil {
		return nil
	}
	campaign := &apiresource.MachineCampaign{
		ScheduleLine:       apiresource.NewEntity(info.ProductionScheduleLineId, constants.ObjectTypeProductionScheduleLine, nil, nil),
		Item:               apiresource.NewEntity(info.ItemId, constants.ObjectTypeItem, nil, &info.Sku),
		SKU:                info.Sku,
		WeekStartDate:      grpcutil.TimestampToTime(info.WeekStartDate),
		WeekIndex:          info.WeekIndex,
		PlannedQuantity:    info.PlannedQuantity,
		ScannedQuantity:    info.ScannedQuantity,
		RemainingQuantity:  info.RemainingQuantity,
		Unit:               info.Unit,
		ReleasedBatchCount: info.ReleasedBatchCount,
		ScannedBatchCount:  info.ScannedBatchCount,
		PlannedRunHours:    info.PlannedRunHours,
		Status:             constants.ProductionScheduleLineStatus(info.StatusCode),
	}
	if info.ProductionRunId != nil && *info.ProductionRunId != "" {
		campaign.ProductionRun = apiresource.NewEntity(*info.ProductionRunId, constants.ObjectTypeProductionRun, nil, nil)
	}
	return campaign
}

func (m *machineStatusSvcImpl) ListMachineStatus(ctx context.Context, req *ListMachineStatusRequest) (*apiresource.List[apiresource.MachineStatus], *apierror.APIError) {
	pbReq := &pb.ListMachineStatusRequest{DepartmentIds: req.DepartmentIDs}
	if req.AsOf != nil {
		pbReq.AsOf = timestamppb.New(*req.AsOf)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, machineStatusSvcTracer, "service.machine_status.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMachineStatusResponse, error) {
			return m.coreClient.ListMachineStatus(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	machines := make([]apiresource.MachineStatus, 0, len(resp.Machines))
	for _, info := range resp.Machines {
		status := apiresource.MachineStatus{
			Object:              constants.ObjectTypeMachineStatus,
			Machine:             apiresource.NewEntity(info.MachineId, constants.ObjectTypeMachine, &info.MachineName, nil),
			Status:              constants.MachineWorkStatus(info.Status),
			Current:             campaignFromProto(info.Current),
			Next:                campaignFromProto(info.Next),
			WeekPlannedQuantity: info.WeekPlannedQuantity,
			WeekScannedQuantity: info.WeekScannedQuantity,
			WeekPlannedRunHours: info.WeekPlannedRunHours,
			Unit:                info.Unit,
		}
		if info.DepartmentId != nil && *info.DepartmentId != "" {
			status.Department = apiresource.NewEntity(*info.DepartmentId, constants.ObjectTypeDepartment, info.DepartmentName, nil)
		}
		if info.Downtime != nil {
			summary := &apiresource.MachineDowntimeSummary{
				Event: apiresource.NewEntity(info.Downtime.EventId, constants.ObjectTypeMachineDowntimeEvent, nil, nil),
				Reason: &apiresource.MachineDowntimeReasonSummary{
					Object: constants.ObjectTypeMachineDowntimeReason,
					Code:   constants.MachineDowntimeReasonCode(info.Downtime.Reason),
				},
				StartedAt: grpcutil.TimestampToTime(info.Downtime.StartedAt),
				Note:      info.Downtime.Note,
			}
			if info.Downtime.ReasonName != "" {
				summary.Reason.Name = &info.Downtime.ReasonName
			}
			if info.Downtime.OeeBucket != "" {
				bucket := constants.OeeBucket(info.Downtime.OeeBucket)
				summary.Reason.OeeBucket = &bucket
			}
			status.Downtime = summary
		}
		machines = append(machines, status)
	}

	// A snapshot of the whole floor, not a page of it: a plant has tens of machines, and paginating a wall display would be worse than useless.
	return apiresource.NewList(machines, apiresource.PageInfo{}), nil
}
