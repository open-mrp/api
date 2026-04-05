package httpgroup

import (
	"fmt"

	roleep "github.com/augno/api/services/api-gateway/endpoints/roles"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type RolesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type RolesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *RolesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("roles endpoint group: core client is required")
	}
	return nil
}

func (*RolesEndpointGroup) Materialize(config *RolesEndpointGroupConfig) *RolesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	roleSvc := roleep.NewRoleSvc(&roleep.RoleSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Roles",
		Description:  "List and manage roles.",
		ResourceType: &apiresource.Role{},
	}

	listRolesEndpoint := (&roleep.ListRolesEndpoint{}).Materialize().WithService(inner, roleSvc)
	getRoleEndpoint := (&roleep.GetRoleEndpoint{}).Materialize().WithService(inner, roleSvc)
	createRoleEndpoint := (&roleep.CreateRoleEndpoint{}).Materialize().WithService(inner, roleSvc)
	updateRoleEndpoint := (&roleep.UpdateRoleEndpoint{}).Materialize().WithService(inner, roleSvc)
	deleteRoleEndpoint := (&roleep.DeleteRoleEndpoint{}).Materialize().WithService(inner, roleSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listRolesEndpoint,
		getRoleEndpoint,
		createRoleEndpoint,
		updateRoleEndpoint,
		deleteRoleEndpoint,
	}

	return &RolesEndpointGroup{inner}
}
