package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// BulkDeleteCustomersRequest is the request to delete multiple customers at once.
type BulkDeleteCustomersRequest struct {
	// The IDs of the customers to delete.
	CustomerIDs []string `json:"customer_ids" validate:"required"`
}

var sampleBulkDeleteCustomersRequest = &BulkDeleteCustomersRequest{
	CustomerIDs: []string{apiresource.SampleCustomerID},
}

func (*BulkDeleteCustomersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkDeleteCustomersRequest)
}

type BulkDeleteCustomersEndpoint struct{}

func (e *BulkDeleteCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkDeleteCustomersRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*BulkDeleteCustomersRequest, *apiresource.EmptyResource]{
		Title:             "Bulk Delete Customers",
		Description:       "Deletes multiple customers in a single operation.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/actions/bulk-delete",
		Request:           &BulkDeleteCustomersRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkDeleteCustomersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(CustomerSvc).BulkDeleteCustomers
		},
	}
}
