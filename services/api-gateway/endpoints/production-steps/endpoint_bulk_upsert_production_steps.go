package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Rate in a bulk upsert operation: a decimal value over a numerator and denominator unit.
type UpsertRateInput struct {
	// Value as a decimal string.
	Value string `json:"value" validate:"required" format:"decimal"`
	// Numerator unit (for example, `$`), referenced by `id`, `name`, or `abbreviation`.
	NumeratorUnit apirequest.UnitIdentifier `json:"numerator_unit" validate:"required"`
	// Denominator unit (for example, `hr`), referenced by `id`, `name`, or `abbreviation`.
	DenominatorUnit apirequest.UnitIdentifier `json:"denominator_unit" validate:"required"`
}

// Production output in a bulk upsert operation.
type UpsertStepProductionInput struct {
	// Item produced by the step, referenced by `id` or `sku`.
	Item apirequest.ItemIdentifier `json:"item" validate:"required"`
	// Quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required" format:"decimal"`
	// Unit for `quantity_value`, referenced by `id`, `name`, or `abbreviation`.
	QuantityUnit apirequest.UnitIdentifier `json:"quantity_unit" validate:"required"`
}

// Material consumption in a bulk upsert operation.
type UpsertStepConsumptionInput struct {
	// Item consumed by the step, referenced by `id` or `sku`.
	Item apirequest.ItemIdentifier `json:"item" validate:"required"`
	// Quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required" format:"decimal"`
	// Unit for `quantity_value`, referenced by `id`, `name`, or `abbreviation`.
	QuantityUnit apirequest.UnitIdentifier `json:"quantity_unit" validate:"required"`
	// Quantity expected to be lost as scrap or waste, as a decimal string. Defaults to zero.
	WasteQuantityValue field.Optional[string] `json:"waste_quantity_value,omitzero" format:"decimal"`
	// Unit for `waste_quantity_value`, referenced by `id`, `name`, or `abbreviation`.
	// Defaults to `quantity_unit`.
	WasteQuantityUnit field.Optional[apirequest.UnitIdentifier] `json:"waste_quantity_unit,omitzero"`
	// Instructions for how this material is consumed.
	Instructions field.Optional[string] `json:"instructions,omitzero"`
}

// Input for a single production step in a bulk upsert operation.
type UpsertProductionStepInput struct {
	// Display name of the step. Rows are matched against existing production steps by
	// name (case-insensitive): a match updates that step, no match creates a new one.
	Name string `json:"name" validate:"required,max=255"`
	// Free-form notes about the step. Preserved when omitted on update.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Leveling correction factor applied to labor time in cost calculations, as a
	// decimal string. Defaults to `0` on create; preserved when omitted on update.
	LevelingFactor field.Optional[string] `json:"leveling_factor,omitzero" format:"decimal"`
	// Allowance correction factor applied to labor time in cost calculations, as a
	// decimal string. Defaults to `0` on create; preserved when omitted on update.
	Allowances field.Optional[string] `json:"allowances,omitzero" format:"decimal"`
	// Scanning station this step reports at, referenced by `id` or `name`. Preserved when
	// omitted on update.
	ScanningStation field.Optional[apirequest.ObjectIdentifier] `json:"scanning_station,omitzero"`
	// Department this step belongs to, referenced by `id` or `name`. Create-only: a row
	// updating an existing step must omit it or state the step's current department.
	Department field.Optional[apirequest.ObjectIdentifier] `json:"department,omitzero"`
	// Cost of labor for this step, expressed as a rate of currency per unit of time
	// (e.g. `$` per `hr`). The numerator unit must be a currency and the denominator
	// must not be. On update a fresh rate is written.
	LaborRate UpsertRateInput `json:"labor_rate" validate:"required"`
	// Labor duration for this step, expressed as a rate (e.g. time per unit of
	// output). On update a fresh rate is written.
	LaborTime UpsertRateInput `json:"labor_time" validate:"required"`
	// Overhead cost for this step, expressed as a rate of currency per unit of time
	// (e.g. `$` per `hr`). The numerator unit must be a currency and the denominator
	// must not be. On update a fresh rate is written.
	OverheadRate UpsertRateInput `json:"overhead_rate" validate:"required"`
	// The item and quantity this step produces. Replaced on update.
	Production UpsertStepProductionInput `json:"production" validate:"required"`
	// Materials consumed by the step. Replaced wholesale on update. Flow DAG edges
	// are derived automatically from item flows: a step consuming an item is linked
	// under the steps producing it.
	Consumptions []UpsertStepConsumptionInput `json:"consumptions" validate:"dive"`
}

// Request to bulk upsert production steps.
type BulkUpsertProductionStepsRequest struct {
	// Production steps to create or update, matched by name (case-insensitive) within
	// the account.
	ProductionSteps []UpsertProductionStepInput `json:"production_steps" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertProductionStepsRequest = &BulkUpsertProductionStepsRequest{
	ProductionSteps: []UpsertProductionStepInput{
		{
			Name:            "Mixing",
			ScanningStation: field.Some(apirequest.ObjectIdentifier{Name: apiresource.SampleScanningStationName}),
			LaborRate:       UpsertRateInput{Value: "25.00", NumeratorUnit: apirequest.UnitIdentifier{Abbreviation: "$"}, DenominatorUnit: apirequest.UnitIdentifier{Abbreviation: "hr"}},
			LaborTime:       UpsertRateInput{Value: "1.5", NumeratorUnit: apirequest.UnitIdentifier{Abbreviation: "hr"}, DenominatorUnit: apirequest.UnitIdentifier{Abbreviation: apiresource.SampleUnitAbbreviation}},
			OverheadRate:    UpsertRateInput{Value: "15.00", NumeratorUnit: apirequest.UnitIdentifier{Abbreviation: "$"}, DenominatorUnit: apirequest.UnitIdentifier{Abbreviation: "hr"}},
			Production:      UpsertStepProductionInput{Item: apirequest.ItemIdentifier{SKU: apiresource.SampleItemSKU}, QuantityValue: "100", QuantityUnit: apirequest.UnitIdentifier{Abbreviation: apiresource.SampleUnitAbbreviation}},
			Consumptions: []UpsertStepConsumptionInput{
				{Item: apirequest.ItemIdentifier{SKU: apiresource.SampleItemSKU}, QuantityValue: "50", QuantityUnit: apirequest.UnitIdentifier{Abbreviation: apiresource.SampleUnitAbbreviation}},
			},
		},
	},
}

func (*BulkUpsertProductionStepsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertProductionStepsRequest)
}

// Creates or updates multiple production steps for the account, matched by name (case-insensitive).
// Validates and resolves synchronously, then writes asynchronously — 202 with a job to poll.
type BulkUpsertProductionStepsEndpoint struct{}

func (e *BulkUpsertProductionStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertProductionStepsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertProductionStepsRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Production Steps",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertProductionStepsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ProductionStepSvc).BulkUpsertProductionSteps
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
