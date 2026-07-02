package supplierep

import (
	"context"
	"net/http"

	"github.com/augno/api/services/auth-service/pkg/types"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// BulkDeleteSuppliersRequest is the request to bulk delete suppliers.
type BulkDeleteSuppliersRequest struct {
	// Supplier IDs to delete.
	SupplierIDs []string `json:"supplier_ids" validate:"required"`
}

var sampleBulkDeleteSuppliersRequest = &BulkDeleteSuppliersRequest{
	SupplierIDs: []string{apiresource.SampleSupplierID},
}

func (*BulkDeleteSuppliersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkDeleteSuppliersRequest)
}

// Deletes multiple suppliers in a single atomic operation.
//
// Each supplier's saved addresses and any users belonging to it are deleted along with it. If any supplier ID is not found, no suppliers are deleted.
type BulkDeleteSuppliersEndpoint struct{}

func (e *BulkDeleteSuppliersEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkDeleteSuppliersRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*BulkDeleteSuppliersRequest, *apiresource.EmptyResource]{
		Title:             "Bulk Delete Suppliers",
		Method:            http.MethodPost,
		Route:             "/v1/operations/suppliers/actions/bulk-delete",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkDeleteSuppliersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SupplierSvc).BulkDeleteSuppliers
		},
	})
}
