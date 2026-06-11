package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete multiple customers.
type BulkDeleteCustomersRequest struct {
	// Customer IDs to delete.
	CustomerIDs []string `json:"customer_ids" validate:"required"`
}

var sampleBulkDeleteCustomersRequest = &BulkDeleteCustomersRequest{
	CustomerIDs: []string{apiresource.SampleCustomerID},
}

func (*BulkDeleteCustomersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkDeleteCustomersRequest)
}

// Deletes multiple customers in a single atomic operation.
//
// Fails with a conflict error if any sales orders still reference any of the customers; if any customer cannot be deleted, none are.
type BulkDeleteCustomersEndpoint struct{}

func (e *BulkDeleteCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkDeleteCustomersRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*BulkDeleteCustomersRequest, *apiresource.EmptyResource]{
		Title:             "Bulk Delete Customers",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/actions/bulk-delete",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkDeleteCustomersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(CustomerSvc).BulkDeleteCustomers
		},
	})
}
