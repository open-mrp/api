package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SamplePaymentTermID = "pytm_skssmsy21lem"
const SamplePaymentTermName = "Net 30"

// When payment is due, such as `Net 30`, assigned to customers and sales orders.
type PaymentTerm struct {
	// Payment term ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=payment_term"`
	// Display name, such as `Net 30`.
	//
	// Unique (case-insensitive) among your account's terms and the system defaults.
	Name string `json:"name" validate:"required"`
	// Whether the term can be assigned to customers and sales orders.
	//
	// `archived` terms stay attached to the records already using them.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// Who owns this payment term.
	//
	// `system` terms are shared defaults and are read-only; `account` terms belong to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Time the payment term was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Time the payment term was last changed.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePaymentTerm = &PaymentTerm{
	ID:        SamplePaymentTermID,
	Object:    constants.ObjectTypePaymentTerm,
	Name:      SamplePaymentTermName,
	Status:    constants.ArchiveStatusActive,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PaymentTerm) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePaymentTerm)
}
