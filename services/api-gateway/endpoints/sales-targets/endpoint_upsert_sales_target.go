package salestargetep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create or update a sales target.
type UpsertSalesTargetRequest struct {
	// ID of the account user (sales rep) the target is for.
	SalesRepID string `path:"id" validate:"required"`
	// ID of the sales target to create or update.
	//
	// If no target with this ID exists, a new one is created with this ID.
	TargetID string `path:"target_id" validate:"required"`
	// Start of the period the target applies to (inclusive).
	//
	// Only applied when creating a new target; the dates on an existing target are not changed.
	StartDate time.Time `json:"start_date"`
	// End of the period the target applies to.
	//
	// Only applied when creating a new target; the dates on an existing target are not changed.
	EndDate time.Time `json:"end_date"`
	// Goal amount for the period, as a decimal string (e.g. `75000.00`).
	AmountValue string `json:"amount_value"`
	// ID of the unit the amount is denominated in (typically a currency unit).
	//
	// Only applied when creating a new target; the unit on an existing target is not changed.
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

// Creates or updates a sales target by ID.
//
// If no target with the given ID exists, one is created with the supplied dates, amount, and unit. If it already exists, only the amount value is updated — the dates and unit are left unchanged.
type UpsertSalesTargetEndpoint struct{}

func (e *UpsertSalesTargetEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpsertSalesTargetRequest, *apiresource.SalesTarget] {
	return (&apiendpoint.APIEndpoint[*UpsertSalesTargetRequest, *apiresource.SalesTarget]{
		Title:             "Upsert Sales Target",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-users/{id}/sales-targets/{target_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesTarget,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSalesTargets, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpsertSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError) {
			return svc.(SalesTargetSvc).UpsertSalesTarget
		},
	})
}
