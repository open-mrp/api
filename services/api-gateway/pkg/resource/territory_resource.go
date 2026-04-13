package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleTerritoryID = "te_01jm4r6700f8nwq3v5hx2d9ktp"

// Sales rep territory assignment.
type Territory struct {
	// Territory ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=territory"`
	// State this territory covers.
	State string `json:"state" validate:"required"`
	// Start of ZIP code range.
	StartZipcode *int32 `json:"start_zipcode"`
	// End of ZIP code range.
	EndZipcode *int32 `json:"end_zipcode"`
	// Sales rep assigned to this territory.
	SalesRep *AccountUser `json:"sales_rep" expandable:"true"`
	// Product line this territory is scoped to.
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
