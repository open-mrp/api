package machineep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func MachinePresenter(m *pb.MachineInfo) apiresource.Machine {
	if m == nil {
		return apiresource.Machine{}
	}

	machine := apiresource.Machine{
		ID:           m.Id,
		Object:       constants.ObjectTypeMachine,
		Name:         m.Name,
		SerialNumber: m.SerialNumber,
		Notes:        m.Notes,
		CreatedAt:    grpcutil.TimestampToTime(m.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(m.UpdatedAt),
	}

	if m.DepartmentId != nil && *m.DepartmentId != "" {
		departmentName := ""
		if m.DepartmentName != nil {
			departmentName = *m.DepartmentName
		}
		machine.Department = &apiresource.Department{
			ID:     *m.DepartmentId,
			Object: constants.ObjectTypeDepartment,
			Name:   departmentName,
		}
	}

	return machine
}

func MachineListPresenter(resp *pb.ListMachinesResponse) *apiresource.List[apiresource.Machine] {
	if resp == nil {
		return apiresource.NewList[apiresource.Machine](nil, apiresource.PageInfo{})
	}

	machines := make([]apiresource.Machine, len(resp.Machines))
	for i, m := range resp.Machines {
		machines[i] = MachinePresenter(m)
	}

	return apiresource.NewList(machines, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
