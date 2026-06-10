package httpgroup

import (
	"fmt"

	permissiongroupep "github.com/augno/api/services/api-gateway/endpoints/permission-groups"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type PermissionGroupsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PermissionGroupsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PermissionGroupsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("permission groups endpoint group: core client is required")
	}
	return nil
}

func (*PermissionGroupsEndpointGroup) Materialize(config *PermissionGroupsEndpointGroupConfig) *PermissionGroupsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	permissionGroupSvc := permissiongroupep.NewPermissionGroupSvc(&permissiongroupep.PermissionGroupSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Permission Groups",
		Description:  "List permission groups and their permissions.",
		ResourceType: &apiresource.PermissionGroup{},
	}

	listPermissionGroupsEndpoint := apiendpoint.From(&permissiongroupep.ListPermissionGroupsEndpoint{}).WithService(inner, permissionGroupSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listPermissionGroupsEndpoint,
	}

	return &PermissionGroupsEndpointGroup{inner}
}
