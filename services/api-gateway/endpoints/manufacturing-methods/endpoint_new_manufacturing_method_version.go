package manufacturingmethodep

import (
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Creates a new draft version of a manufacturing method.
//
// The draft copies the method's operations and materials and takes the item's next version number. Edit it, then activate it to replace the active version.
type NewManufacturingMethodVersionEndpoint struct{}

func (e *NewManufacturingMethodVersionEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.ManufacturingMethod] {
	ep := lifecycle.Action("Create Manufacturing Method Version", methodsRoute+"/{id}/actions/new-version", constants.ObjectTypeManufacturingMethod,
		authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionCreate}}},
		func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.ManufacturingMethod] {
			return svc.(ManufacturingMethodSvc).NewManufacturingMethodVersion
		})
	ep.IncludeConfig = methodIncludes()
	return ep
}
