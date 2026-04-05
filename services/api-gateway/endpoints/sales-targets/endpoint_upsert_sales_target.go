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

// UpsertSalesTargetRequest is the request to create or update a sales target.
type UpsertSalesTargetRequest struct {
	// The user ID (sales rep) for the target.
	SalesRepID string `path:"id" validate:"required"`
	// The sales target ID to create or update.
	TargetID string `path:"target_id" validate:"required"`
	// The start date for the sales target.
	StartDate time.Time `json:"start_date"`
	// The end date for the sales target.
	EndDate time.Time `json:"end_date"`
	// The target amount value (decimal string).
	AmountValue string `json:"amount_value"`
	// The unit ID for the target amount.
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
