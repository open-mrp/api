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

// Request to create or update a sales target.
type UpsertSalesTargetRequest struct {
	// Sales rep user ID.
	SalesRepID string `path:"id" validate:"required"`
	// Sales target ID.
	TargetID string `path:"target_id" validate:"required"`
	// Start date.
	StartDate time.Time `json:"start_date"`
	// End date.
	EndDate time.Time `json:"end_date"`
	// Target amount value (decimal string).
	AmountValue string `json:"amount_value"`
	// Amount unit ID.
	AmountUnitID string `json:"amount_unit_id"`
}

var sampleUpsertSalesTargetRequest = &UpsertSalesTargetRequest{
	StartDate:    time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	EndDate:      time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	AmountValue:  "75000.00",
	AmountUnitID: apiresource.SampleUnitID,
}

func (*UpsertSalesTargetRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpsertSalesTargetRequest)
}

type UpsertSalesTargetEndpoint struct{}

func (e *UpsertSalesTargetEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpsertSalesTargetRequest, *apiresource.SalesTarget] {
	return &apiendpoint.APIEndpoint[*UpsertSalesTargetRequest, *apiresource.SalesTarget]{
		Title:             "Upsert Sales Target",
		Description:       "Creates or updates a sales target by ID.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-users/{id}/sales-targets/{target_id}",
		Request:           &UpsertSalesTargetRequest{},
		Response:          &apiresource.SalesTarget{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpsertSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError) {
			return svc.(SalesTargetSvc).UpsertSalesTarget
		},
	}
}
