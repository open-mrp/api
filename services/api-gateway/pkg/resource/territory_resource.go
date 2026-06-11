package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleTerritoryID = "te_0132f802e5603f7d356fac79d1"

// A geographic sales region that assigns a sales rep to a state or ZIP code range.
//
// When a sales order is created without an explicit sales rep, territories are used to auto-assign one from the order's ship-to address: the customer's default sales rep takes precedence, then a territory matching the ship-to ZIP code, then a territory covering the entire ship-to state.
type Territory struct {
	// Territory ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=territory"`
	// State this territory covers (e.g. `NY`).
	State string `json:"state" validate:"required"`
	// Inclusive start of the ZIP code range this territory covers within the state.
	//
	// Unset when the territory spans the entire state rather than a ZIP code range.
	StartZipcode *int32 `json:"start_zipcode"`
	// Inclusive end of the ZIP code range this territory covers within the state.
	//
	// Unset when the territory spans the entire state rather than a ZIP code range.
	EndZipcode *int32 `json:"end_zipcode"`
	// Sales rep (account user) assigned to orders matching this territory.
	SalesRep *AccountUser `json:"sales_rep" expandable:"true"`
	// Product line this territory is scoped to.
	//
	// Unset when the territory matches orders regardless of product line.
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
