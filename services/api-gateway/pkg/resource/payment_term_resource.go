package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePaymentTermID = "pytm_skssmsy21lem"
const SamplePaymentTermName = "Net 30"

// A payment term describing when payment is due (e.g. `Net 30`), assignable to customers, sales orders, purchase orders, and invoices.
type PaymentTerm struct {
	// Payment term ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=payment_term"`
	// Display name (e.g. `Net 30`), unique among the payment terms visible to your account.
	Name string `json:"name" validate:"required"`
	// Whether this payment term is still in active use.
	//
	// Payment terms created through the API are always `active`, and no endpoint changes a term's status. List Payment Terms returns inactive terms alongside active ones, so filter them out yourself if you only want the ones still on offer.
	Status constants.PaymentTermStatus `json:"status" validate:"required"`
	// Provenance of this payment term.
	//
	// System-owned payment terms are platform-provided defaults shared across all accounts and cannot be updated or deleted; account-owned payment terms are custom to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePaymentTerm = &PaymentTerm{
	ID:        SamplePaymentTermID,
	Object:    constants.ObjectTypePaymentTerm,
	Name:      SamplePaymentTermName,
	Status:    constants.PaymentTermStatusActive,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PaymentTerm) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePaymentTerm)
}
