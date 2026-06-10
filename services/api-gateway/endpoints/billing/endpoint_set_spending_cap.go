package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to set or remove the monthly spending cap.
type SetSpendingCapRequest struct {
	// Monthly spending cap in cents. Null removes the cap; omitting the
	// field leaves the current cap unchanged.
	CapCents field.Clearable[int64] `json:"cap_cents,omitzero"`
}

var sampleSetSpendingCapRequest = &SetSpendingCapRequest{
	CapCents: field.Set(int64(50000)),
}

func (*SetSpendingCapRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetSpendingCapRequest)
}

// Sets or removes the monthly agent spending cap for the account.
type SetSpendingCapEndpoint struct{}

func (e *SetSpendingCapEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetSpendingCapRequest, *apiresource.SpendingCapResponse] {
	return (&apiendpoint.APIEndpoint[*SetSpendingCapRequest, *apiresource.SpendingCapResponse]{
		Title:             "Set Spending Cap",
		Method:            http.MethodPut,
		Route:             "/v1/billing/spending-cap",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSpendingCapResponse,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetSpendingCapRequest) (*apiresource.SpendingCapResponse, *apierror.APIError) {
			return svc.(BillingSvc).SetSpendingCap
		},
	})
}
