package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpsertUnitInput is the input for a single unit in a bulk upsert operation.
type UpsertUnitInput struct {
	// Display name of the unit (e.g. "Gram"). A row matching a system unit fails — system
	// units cannot be modified.
	Name string `json:"name" validate:"required,max=255"`
	// Short abbreviation for the unit (e.g. "g"). Also used for matching — see `name`.
	Abbreviation string `json:"abbreviation" validate:"required,max=191"`
	// Unit dimension code. Create-only — a row that changes an existing unit's dimension fails.
	Type constants.UnitType `json:"type" validate:"required"`
	// Conversion ratio numerator relative to the base unit, as a decimal string.
	RatioNumerator string `json:"ratio_numerator" validate:"required" format:"decimal"`
	// Conversion ratio denominator relative to the base unit, as a decimal string.
	RatioDenominator string `json:"ratio_denominator" validate:"required" format:"decimal"`
	// Conversion offset numerator, as a decimal string.
	OffsetNumerator string `json:"offset_numerator" validate:"required" format:"decimal"`
	// Conversion offset denominator, as a decimal string.
	OffsetDenominator string `json:"offset_denominator" validate:"required" format:"decimal"`
	// Whether the unit is its dimension's base unit. Bulk upsert never creates a base unit
	// and rejects a change to an existing one.
	IsBaseUnit bool `json:"is_base_unit"`
}

// BulkUpsertUnitsRequest is the request to bulk upsert units.
type BulkUpsertUnitsRequest struct {
	// Units to create or update, matched by name or abbreviation within the account.
	Units []UpsertUnitInput `json:"units" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertUnitsRequest = &BulkUpsertUnitsRequest{
	Units: []UpsertUnitInput{
		{
			Name:              apiresource.SampleUnitName,
			Abbreviation:      apiresource.SampleUnitAbbreviation,
			Type:              constants.UnitTypeMass,
			RatioNumerator:    "1000",
			RatioDenominator:  "1",
			OffsetNumerator:   "0",
			OffsetDenominator: "1",
			IsBaseUnit:        false,
		},
	},
}

func (*BulkUpsertUnitsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertUnitsRequest)
}

// Creates or updates multiple units of measure for the account, matched by name or
// abbreviation, then writes asynchronously — 202 with a job to poll.
type BulkUpsertUnitsEndpoint struct{}

func (e *BulkUpsertUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertUnitsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertUnitsRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Units",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/units/actions/bulk-upsert",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertUnitsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(UnitSvc).BulkUpsertUnits
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
