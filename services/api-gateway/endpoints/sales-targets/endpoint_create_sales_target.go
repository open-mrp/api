package salestargetep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a sales target.
type CreateSalesTargetRequest struct {
	// Sales rep user ID.
	SalesRepID string `path:"id" validate:"required"`
	// Start date.
	StartDate time.Time `json:"start_date"`
	// End date.
	EndDate time.Time `json:"end_date"`
	// Target amount value (decimal string).
	AmountValue string `json:"amount_value"`
	// Amount unit ID.
	AmountUnitID string `json:"amount_unit_id" validate:"max=191"`
}

var sampleCreateSalesTargetRequest = &CreateSalesTargetRequest{
	StartDate:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	EndDate:      time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
	AmountValue:  "50000.00",
	AmountUnitID: apiresource.SampleUnitID,
}

func (*CreateSalesTargetRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSalesTargetRequest)
}

type CreateSalesTargetEndpoint struct{}

func (e *CreateSalesTargetEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesTargetRequest, *apiresource.SalesTarget] {
	return &apiendpoint.APIEndpoint[*CreateSalesTargetRequest, *apiresource.SalesTarget]{
		Title:             "Create Sales Target",
		Description:       "Creates a sales target for an account user.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-users/{id}/sales-targets",
		Request:           &CreateSalesTargetRequest{},
		Response:          &apiresource.SalesTarget{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError) {
			return svc.(SalesTargetSvc).CreateSalesTarget
		},
	}
}
