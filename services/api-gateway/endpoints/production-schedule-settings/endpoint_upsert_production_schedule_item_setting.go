package productionschedulesettingsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to write one item's planning overrides.
type UpsertItemSettingRequest struct {
	// Item to override.
	ItemID string `path:"item_id" validate:"required"`
	// Whether this item takes part in planning.
	//
	// An excluded item is left out of the plan entirely: no campaigns, no policy, no capacity.
	ParticipationStatus constants.ParticipationStatus `json:"participation_status" validate:"required"`
	// Units in one production lot for this item, overriding the lot its product line would supply.
	LotMultipleUnits field.Optional[float64] `json:"lot_multiple_units,omitzero" validate:"omitempty,gt=0"`
	// How this item is produced.
	//
	// - `make_to_stock`: built to the forecast, holding a safety stock against its variability.
	// - `make_to_order`: built only against orders already on the book, holding no buffer.
	//
	// Clearing it returns the item to its product line's policy, then to the account default.
	FulfillmentPolicy field.Clearable[constants.FulfillmentPolicy] `json:"fulfillment_policy,omitzero"`
}

var sampleUpsertItemSettingRequest = &UpsertItemSettingRequest{
	ItemID:              apiresource.SampleItemID,
	ParticipationStatus: constants.ParticipationStatusIncluded,
	FulfillmentPolicy:   field.Set(constants.FulfillmentPolicyMakeToOrder),
}

func (*UpsertItemSettingRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpsertItemSettingRequest)
}

// Writes the planning overrides for one item.
//
// An item has at most one set of overrides, so this replaces the existing entry rather than adding a second, and the entry keeps the ID it already had.
//
// The fulfillment policy is the most consequential of these. A `make_to_order` item contributes no forecast demand and holds no safety stock, so it is built only against orders already on the book — which is what stops a slow mover accumulating inventory nobody asked for. It also propagates: an intermediate item is planned to order only when every finished good it becomes is, so one stocked sibling keeps the whole family buffered.
//
// Overrides are read when a plan is generated, so a change takes effect on the next generated version and leaves existing ones untouched.
type UpsertItemSettingEndpoint struct{}

func (e *UpsertItemSettingEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpsertItemSettingRequest, *apiresource.ProductionScheduleItemSetting] {
	return (&apiendpoint.APIEndpoint[*UpsertItemSettingRequest, *apiresource.ProductionScheduleItemSetting]{
		Title:             "Upsert Production Schedule Item Setting",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedule-settings/items/{item_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionScheduleItemSetting,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpsertItemSettingRequest) (*apiresource.ProductionScheduleItemSetting, *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).UpsertItemSetting
		},
	})
}
