package customerep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to merge source customers into a target customer.
type MergeCustomersRequest struct {
	// ID of the customer the others are merged into.
	CustomerID string `path:"id" validate:"required"`
	// IDs of the customers to merge into this one. They are deleted once merged; no duplicates, and not this customer's ID.
	SourceCustomerIDs []string `json:"source_customer_ids" validate:"required"`
}

var sampleMergeCustomersRequest = &MergeCustomersRequest{
	SourceCustomerIDs: []string{apiresource.SampleCustomerID},
}

func (*MergeCustomersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMergeCustomersRequest)
}

// Merges customers into this one.
//
// Orders, invoices, shipments and deliveries of the source customers move to this customer. Group memberships, product line access, addresses and users are combined; the source customers' own price lists are discarded, and the sources' child customers move under this one. Notification recipients of the sources are discarded. This customer keeps its own name, number and defaults. The source customers are deleted. Runs in the background; the response is a job to poll.
type MergeCustomersEndpoint struct{}

func (e *MergeCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*MergeCustomersRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*MergeCustomersRequest, *apiresource.AsyncJob]{
		Title:             "Merge Customers",
		Method:            http.MethodPost,
		Route:             "/v1/sales/customers/{id}/actions/merge",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		// Merging rewrites the target and deletes the sources, so it takes both.
		Auth:       authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionDelete}}, RequireAll: true},
		ObjectType: constants.ObjectTypeAsyncJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *MergeCustomersRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(CustomerSvc).MergeCustomers
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
