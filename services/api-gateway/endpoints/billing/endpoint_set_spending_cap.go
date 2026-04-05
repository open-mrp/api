package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type SetSpendingCapRequest struct {
	// The monthly spending cap in cents. Null to remove the cap.
	CapCents *int64 `json:"cap_cents"`
}

var sampleSpendingCapCents = int64(50000)

var sampleSetSpendingCapRequest = &SetSpendingCapRequest{
	CapCents: &sampleSpendingCapCents,
}

func (*SetSpendingCapRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetSpendingCapRequest)
}

type SetSpendingCapEndpoint struct{}

func (e *SetSpendingCapEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetSpendingCapRequest, *apiresource.SpendingCapResponse] {
	return &apiendpoint.APIEndpoint[*SetSpendingCapRequest, *apiresource.SpendingCapResponse]{
		Title:             "Set Spending Cap",
		Description:       "Sets or removes the monthly agent spending cap for the account. Pass null cap_cents to remove the cap.",
		Method:            http.MethodPut,
		Route:             "/v1/billing/spending-cap",
		ContentType:       "application/json",
		Request:           &SetSpendingCapRequest{},
		Response:          &apiresource.SpendingCapResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetSpendingCapRequest) (*apiresource.SpendingCapResponse, *apierror.APIError) {
			return svc.(BillingSvc).SetSpendingCap
		},
	}
}
