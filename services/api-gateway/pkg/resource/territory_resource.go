package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleTerritoryID = "te_gfs3vr2jpwgm"

// A geographic sales region that assigns a sales rep to a state or ZIP code range.
//
// When a sales order is created without an explicit sales rep, one is auto-assigned: the customer's default sales rep takes precedence, then a territory matching the ship-to address's ZIP code, then a territory covering the entire ship-to state.
//
// Territories are skipped entirely when the customer is commission-exempt or every line on the order belongs to a commission-exempt product line; those orders are left without a sales rep.
type Territory struct {
	// Territory ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=territory"`
	// State this territory covers (e.g. `NY`).
	//
	// The state is only used to match orders when the territory has no ZIP code range; territories with a ZIP code range are matched on the ZIP code alone. Matching is an exact comparison against the ship-to address's state, so use the same format your addresses use.
	State string `json:"state" validate:"required"`
	// Inclusive start of the ZIP code range this territory covers.
	//
	// Unset when the territory spans the entire state rather than a ZIP code range.
	StartZipcode *int32 `json:"start_zipcode"`
	// Inclusive end of the ZIP code range this territory covers.
	//
	// A territory with a start ZIP code but no end ZIP code matches that single ZIP code.
	EndZipcode *int32 `json:"end_zipcode"`
	// Account user credited as the sales rep on orders matching this territory.
	SalesRep *AccountUser `json:"sales_rep" expandable:"true"`
	// Product line this territory is associated with.
	//
	// Sales rep auto-assignment matches on ZIP code and state only, so this records what the territory covers rather than narrowing which orders it matches.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// When this territory was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this territory was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleTerritoryStartZipcode int32 = 10001
var sampleTerritoryEndZipcode int32 = 10999

var SampleTerritory = &Territory{
	ID:           SampleTerritoryID,
	Object:       constants.ObjectTypeTerritory,
	State:        "NY",
	StartZipcode: &sampleTerritoryStartZipcode,
	EndZipcode:   &sampleTerritoryEndZipcode,
	SalesRep:     nil,
	ProductLine:  nil,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Territory) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTerritory)
}
