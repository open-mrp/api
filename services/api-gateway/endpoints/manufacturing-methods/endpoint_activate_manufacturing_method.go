package manufacturingmethodep

import (
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Activates a draft manufacturing method, making it the method for new production of its item.
//
// The item's previously active method becomes `archived`. The method needs at least one operation, each with a work center. Only a `draft` can be activated; any other status is a 409 `invalid_state_transition`.
type ActivateManufacturingMethodEndpoint struct{}

func (e *ActivateManufacturingMethodEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.ManufacturingMethod] {
	ep := lifecycle.Action("Activate Manufacturing Method", methodsRoute+"/{id}/actions/activate", constants.ObjectTypeManufacturingMethod,
		authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionUpdate}}},
		func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.ManufacturingMethod] {
			return svc.(ManufacturingMethodSvc).ActivateManufacturingMethod
		})
	ep.IncludeConfig = methodIncludes()
	return ep
}
