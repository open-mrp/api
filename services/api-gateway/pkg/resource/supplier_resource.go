package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

// An account you buy from.
//
// A supplier is another account in a selling relationship with yours, so it is referenced from purchase orders, receiving orders and deliveries as well as retrieved on its own. Everything past its identity is expandable or nullable, because a supplier named from one of those documents is known by id, name and number alone.
type Supplier struct {
	// Supplier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier"`
	// The supplier's name, as shown in the dashboard and on documents.
	Name string `json:"name" validate:"required"`
	// Human-facing supplier code, unique per account (e.g. `SUP-001`).
	Number string `json:"number" validate:"required"`
	// Free-form notes about the supplier.
	Note *string `json:"note"`
	// The supplier's default billing address.
	//
	// A new address can be created inline when the supplier is created; afterwards this default is changed by passing `bill_to_address_id` to the update endpoint.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// The supplier's default shipping address.
	//
	// When a supplier is created with only a bill-to address, that same address also becomes the default shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// Number of materials sourced from this supplier.
	//
	// Counts every material linked to the supplier, including links whose status is `inactive`. Zero on a supplier named from another document, which does not count them.
	MaterialCount int64 `json:"material_count"`
	// Creation timestamp.
	//
	// Null on a supplier named from another document, which carries its identity rather than its record.
	CreatedAt *time.Time `json:"created_at"`
	// Last updated timestamp.
	UpdatedAt *time.Time `json:"updated_at"`
}

var sampleSupplierCreatedAt = timeutil.TimestampToTime(sampleCreatedAtTimestamp)
var sampleSupplierUpdatedAt = timeutil.TimestampToTime(sampleUpdatedAtTimestamp)

var SampleSupplier = &Supplier{
	ID:            SampleSupplierID,
	Object:        constants.ObjectTypeSupplier,
	Name:          SampleSupplierName,
	Number:        SampleSupplierNumber,
	BillToAddress: SampleAddress,
	ShipToAddress: SampleAddress,
	MaterialCount: 5,
	CreatedAt:     &sampleSupplierCreatedAt,
	UpdatedAt:     &sampleSupplierUpdatedAt,
}

func (*Supplier) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupplier)
}
