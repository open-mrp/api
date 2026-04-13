package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type GetSpendingCapEndpoint struct{}

func (e *GetSpendingCapEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.SpendingCapResponse] {
	return &apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.SpendingCapResponse]{
		Title:             "Get Spending Cap",
		Description:       "Returns the monthly agent spending cap for the account. Null cap_cents means no cap.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/billing/spending-cap",
		Request:           &apiresource.EmptyResource{},
		Response:          &apiresource.SpendingCapResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.SpendingCapResponse, *apierror.APIError) {
			return svc.(BillingSvc).GetSpendingCap
		},
	}
}
