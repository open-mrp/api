package httpgroup

import (
	"fmt"

	departmentep "github.com/augno/api/services/api-gateway/endpoints/departments"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type DepartmentsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type DepartmentsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *DepartmentsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("departments endpoint group: core client is required")
	}
	return nil
}

func (*DepartmentsEndpointGroup) Materialize(config *DepartmentsEndpointGroupConfig) *DepartmentsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	departmentSvc := departmentep.NewDepartmentSvc(&departmentep.DepartmentSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Departments Management",
		Description:  "List and manage departments.",
		ResourceType: &apiresource.Department{},
	}

	listDepartmentsEndpoint := (&departmentep.ListDepartmentsEndpoint{}).Materialize().WithService(inner, departmentSvc)
	getDepartmentEndpoint := (&departmentep.RetrieveDepartmentEndpoint{}).Materialize().WithService(inner, departmentSvc)
	createDepartmentEndpoint := (&departmentep.CreateDepartmentEndpoint{}).Materialize().WithService(inner, departmentSvc)
	updateDepartmentEndpoint := (&departmentep.UpdateDepartmentEndpoint{}).Materialize().WithService(inner, departmentSvc)
	deleteDepartmentEndpoint := (&departmentep.DeleteDepartmentEndpoint{}).Materialize().WithService(inner, departmentSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listDepartmentsEndpoint,
		getDepartmentEndpoint,
		createDepartmentEndpoint,
		updateDepartmentEndpoint,
		deleteDepartmentEndpoint,
	}

	return &DepartmentsEndpointGroup{inner}
}
