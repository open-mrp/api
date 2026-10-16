package supplierep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list suppliers.
type ListSuppliersRequest struct {
	object.ListRequest
	apirequest.CreatedRange
	// Case-insensitive substring match on `name` or `number`. Each word must match on its own.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return suppliers with any of these statuses: `active` or `archived`.
	Statuses []constants.ArchiveStatus `query:"statuses"`
	// Only return suppliers with a supplier item for any of these items, active or archived.
	ItemIDs []string `query:"item_ids"`
}

// Returns a list of suppliers.
//
// Sorted by creation date, newest first.
type ListSuppliersEndpoint struct{}

func (e *ListSuppliersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSuppliersRequest, *apiresource.List[apiresource.Supplier]] {
	return (&apiendpoint.APIEndpoint[*ListSuppliersRequest, *apiresource.List[apiresource.Supplier]]{
		Title:             "List Suppliers",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSupplier,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSuppliersRequest) (*apiresource.List[apiresource.Supplier], *apierror.APIError) {
			return svc.(SupplierSvc).ListSuppliers
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplier,
			Fields:     supplierIncludeFields,
		}),
	})
}
