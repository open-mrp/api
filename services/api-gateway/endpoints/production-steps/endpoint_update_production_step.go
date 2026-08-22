package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a production step.
type UpdateProductionStepRequest struct {
	// Production step ID.
	ProductionStepID string `path:"id" validate:"required"`
	// New display name.
	//
	// Must be unique within the account.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Leveling correction factor applied to labor time in cost calculations, as a decimal string.
	LevelingFactor field.Optional[string] `json:"leveling_factor,omitzero"`
	// Allowance correction factor applied to labor time in cost calculations, as a decimal string.
	Allowances field.Optional[string] `json:"allowances,omitzero"`
	// Scanning station where batches at this step are scanned.
	ScanningStationID field.Optional[string] `json:"scanning_station_id,omitzero" validate:"omitempty"`
}

var sampleUpdateProductionStepName = "Assembly Step A"
var sampleUpdateProductionStepLevelingFactor = "1.15"
var sampleUpdateProductionStepScanningStationID = apiresource.SampleScanningStationID
var sampleUpdateProductionStepRequest = &UpdateProductionStepRequest{
	Name:              field.Some(sampleUpdateProductionStepName),
	LevelingFactor:    field.Some(sampleUpdateProductionStepLevelingFactor),
	ScanningStationID: field.Some(sampleUpdateProductionStepScanningStationID),
}

func (*UpdateProductionStepRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductionStepRequest)
}

// Partially updates a production step.
//
// Returns a conflict error if another production step in the account already uses the requested name.
type UpdateProductionStepEndpoint struct{}

func (e *UpdateProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionStepRequest, *apiresource.ProductionStep] {
	return (&apiendpoint.APIEndpoint[*UpdateProductionStepRequest, *apiresource.ProductionStep]{
		Title:               "Update Production Step",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/operations/production-steps/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionSteps, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeProductionStep,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
			return svc.(ProductionStepSvc).UpdateProductionStep
		},
	})
}
