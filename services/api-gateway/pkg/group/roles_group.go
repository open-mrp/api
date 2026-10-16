package httpgroup

import (
	"fmt"

	roleep "github.com/open-mrp/api/services/api-gateway/endpoints/roles"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type RolesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
	// Permissions and Templates are the permission catalog and role templates, served from the role service.
	Permissions *apiendpoint.APIEndpointGroup
	Templates   *apiendpoint.APIEndpointGroup
}

type RolesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
		Description:  "What people, devices, API keys and agents may do.",
		ResourceType: &apiresource.Role{},
	}

	listRolesEndpoint := apiendpoint.From(&roleep.ListRolesEndpoint{}).WithService(inner, roleSvc)
	getRoleEndpoint := apiendpoint.From(&roleep.RetrieveRoleEndpoint{}).WithService(inner, roleSvc)
	createRoleEndpoint := apiendpoint.From(&roleep.CreateRoleEndpoint{}).WithService(inner, roleSvc)
	updateRoleEndpoint := apiendpoint.From(&roleep.UpdateRoleEndpoint{}).WithService(inner, roleSvc)
	deleteRoleEndpoint := apiendpoint.From(&roleep.DeleteRoleEndpoint{}).WithService(inner, roleSvc)
	previewRoleEndpoint := apiendpoint.From(&roleep.PreviewRoleEndpoint{}).WithService(inner, roleSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listRolesEndpoint,
		getRoleEndpoint,
		createRoleEndpoint,
		updateRoleEndpoint,
		deleteRoleEndpoint,
		previewRoleEndpoint,
	}

	permissions := &apiendpoint.APIEndpointGroup{
		Title:        "Permissions",
		Description:  "The permission catalog roles are built from.",
		ResourceType: &apiresource.Permission{},
	}
	permissions.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&roleep.ListPermissionsEndpoint{}).WithService(permissions, roleSvc),
	}

	templates := &apiendpoint.APIEndpointGroup{
		Title:        "Role Templates",
		Description:  "Starting points for roles.",
		ResourceType: &apiresource.RoleTemplate{},
	}
	templates.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&roleep.ListRoleTemplatesEndpoint{}).WithService(templates, roleSvc),
	}

	return &RolesEndpointGroup{APIEndpointGroup: inner, Permissions: permissions, Templates: templates}
}
