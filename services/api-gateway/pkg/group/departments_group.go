package httpgroup

import (
	"fmt"

	departmentep "github.com/open-mrp/api/services/api-gateway/endpoints/departments"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type DepartmentsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type DepartmentsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Departments",
		Description:  "List and manage departments.",
		ResourceType: &apiresource.Department{},
	}

	listDepartmentsEndpoint := apiendpoint.From(&departmentep.ListDepartmentsEndpoint{}).WithService(inner, departmentSvc)
	getDepartmentEndpoint := apiendpoint.From(&departmentep.RetrieveDepartmentEndpoint{}).WithService(inner, departmentSvc)
	createDepartmentEndpoint := apiendpoint.From(&departmentep.CreateDepartmentEndpoint{}).WithService(inner, departmentSvc)
	updateDepartmentEndpoint := apiendpoint.From(&departmentep.UpdateDepartmentEndpoint{}).WithService(inner, departmentSvc)
	deleteDepartmentEndpoint := apiendpoint.From(&departmentep.DeleteDepartmentEndpoint{}).WithService(inner, departmentSvc)
	bulkUpsertDepartmentsEndpoint := apiendpoint.From(&departmentep.BulkUpsertDepartmentsEndpoint{}).WithService(inner, departmentSvc)
	exportDepartmentsEndpoint := apiendpoint.From(&departmentep.ExportDepartmentsEndpoint{}).WithService(inner, departmentSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listDepartmentsEndpoint,
		getDepartmentEndpoint,
		createDepartmentEndpoint,
		updateDepartmentEndpoint,
		deleteDepartmentEndpoint,
		bulkUpsertDepartmentsEndpoint,
		exportDepartmentsEndpoint,
	}

	return &DepartmentsEndpointGroup{inner}
}
