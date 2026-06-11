package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePaymentTermID = "pytm_018694d6601ea771cd1b52e890"
const SamplePaymentTermName = "Net 30"

// Payment term resource.
type PaymentTerm struct {
	// Payment term ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=payment_term"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Payment term status.
	//
	// - `active`: the term is available for assignment to customers and invoices.
	// - `inactive`: the term is retained for historical records but cannot be assigned.
	Status constants.PaymentTermStatus `json:"status" validate:"required"`
	// Owner of this resource, indicating whether it is an Augno-provided system default or was created by your account.
	//
	// Expandable via include[]=owner.
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
