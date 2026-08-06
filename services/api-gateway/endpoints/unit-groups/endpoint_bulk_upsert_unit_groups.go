package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpsertUnitGroupConversionInput is the input for a single unit conversion within a bulk upsert unit group.
type UpsertUnitGroupConversionInput struct {
	// Unit for this conversion, referenced by `id`, `name`, or `abbreviation`.
	Unit apirequest.UnitIdentifier `json:"unit" validate:"required"`
	// Discount percentage to apply for this unit conversion.
	DiscountPercentage *float64 `json:"discount_percentage,omitempty" default:"1" nullable:"false"`
}

// UpsertUnitGroupInput is the input for a single unit group in a bulk upsert operation.
type UpsertUnitGroupInput struct {
	// Display name of the unit group, matched case-insensitively against existing groups.
	// A row matching a system unit group fails — system groups cannot be modified.
	Name string `json:"name" validate:"required,max=255"`
	// Free-form notes about the unit group. Preserved when omitted on update.
	Notes *string `json:"notes,omitempty" default:"null" nullable:"false"`
	// Unit dimension type. Create-only — an existing group keeps its stored type.
	Type constants.UnitType `json:"type" validate:"required"`
	// Base unit for this group, referenced by `id`, `name`, or `abbreviation`.
	BaseUnit apirequest.UnitIdentifier `json:"base_unit" validate:"required"`
	// Units to associate with the group. Replaces the existing set on update; the base
	// unit is always kept.
	UnitConversions []UpsertUnitGroupConversionInput `json:"unit_conversions,omitempty"`
}

// BulkUpsertUnitGroupsRequest is the request to bulk upsert unit groups.
type BulkUpsertUnitGroupsRequest struct {
	// Unit groups to create or update, matched by name within the account.
	UnitGroups []UpsertUnitGroupInput `json:"unit_groups" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertUnitGroupDiscountPct = float64(1)
var sampleBulkUpsertUnitGroupsRequest = &BulkUpsertUnitGroupsRequest{
	UnitGroups: []UpsertUnitGroupInput{
		{
			Name:     apiresource.SampleUnitGroupName,
			Type:     constants.UnitTypeMass,
			BaseUnit: apirequest.UnitIdentifier{ID: apiresource.SampleUnitID},
			UnitConversions: []UpsertUnitGroupConversionInput{
				{
					Unit:               apirequest.UnitIdentifier{ID: apiresource.SampleUnitID},
					DiscountPercentage: &sampleBulkUpsertUnitGroupDiscountPct,
				},
			},
		},
	},
}

func (*BulkUpsertUnitGroupsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertUnitGroupsRequest)
}

// Creates or updates multiple unit groups for the account, matched by name
// (case-insensitive), then writes asynchronously — 202 with a job to poll.
type BulkUpsertUnitGroupsEndpoint struct{}

func (e *BulkUpsertUnitGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertUnitGroupsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertUnitGroupsRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Unit Groups",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/unit-groups/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertUnitGroupsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(UnitGroupSvc).BulkUpsertUnitGroups
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
