package unitep

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

// One unit to create or update, matched to an existing unit by `abbreviation`.
type UpsertUnitInput struct {
	// Display name, such as `Kilogram`.
	//
	// Unique (case-insensitive) among your account's units and the system units; a name another unit has fails the row with `resource_exists`.
	Name string `json:"name" validate:"required,max=255"`
	// Short label, such as `kg`. Matches the row to an existing unit (case-insensitive).
	//
	// A row matching a system unit fails, since system units can't be updated.
	Abbreviation string `json:"abbreviation" validate:"required,max=32"`
	// What the unit measures. A row that changes an existing unit's dimension fails.
	Dimension constants.UnitDimension `json:"dimension" validate:"required"`
	// The ratio that converts a value in this unit to its dimension's base unit.
	RatioNumerator string `json:"ratio_numerator" validate:"required" format:"decimal"`
	// The ratio that converts a value in this unit to its dimension's base unit. Can't be zero.
	RatioDenominator string `json:"ratio_denominator" validate:"required" format:"decimal"`
	// An offset added after the ratio, for scales with different zero points, such as temperature.
	OffsetNumerator string `json:"offset_numerator" validate:"required" format:"decimal"`
	// An offset added after the ratio, for scales with different zero points. Can't be zero.
	OffsetDenominator string `json:"offset_denominator" validate:"required" format:"decimal"`
}

// BulkUpsertUnitsRequest is the request to bulk upsert units.
type BulkUpsertUnitsRequest struct {
	// Units to create or update, at most 1,000.
	Units []UpsertUnitInput `json:"units" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertUnitsRequest = &BulkUpsertUnitsRequest{
	Units: []UpsertUnitInput{
		{
			Name:              apiresource.SampleUnitName,
			Abbreviation:      apiresource.SampleUnitAbbreviation,
			Dimension:         constants.UnitDimensionMass,
			RatioNumerator:    "1000",
			RatioDenominator:  "1",
			OffsetNumerator:   "0",
			OffsetDenominator: "1",
		},
	},
}

func (*BulkUpsertUnitsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertUnitsRequest)
}

// Creates or updates units in bulk.
//
// Each row is matched to an existing unit by `abbreviation` (case-insensitive). Unmatched rows create new units. The work runs in the background; the response is an async job to poll.
type BulkUpsertUnitsEndpoint struct{}

func (e *BulkUpsertUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertUnitsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertUnitsRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Upsert Units",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/units/actions/bulk-upsert",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionCreate}, {Domain: types.PermissionDomainUnits, Action: types.ActionUpdate}}, RequireAll: true},
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertUnitsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(UnitSvc).BulkUpsertUnits
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
