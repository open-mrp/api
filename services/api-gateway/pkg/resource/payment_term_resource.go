package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePaymentTermID = "pytm_01jm4r6700f8nwq3v5hx2d9ktp"
const SamplePaymentTermName = "Net 30"

// PaymentTerm represents an account-owned or default payment term.
type PaymentTerm struct {
	// The unique identifier for the payment term.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=payment_term"`
	// The display name of the payment term.
	Name string `json:"name" validate:"required"`
	// The status of the payment term.
	Status constants.PaymentTermStatus `json:"status" validate:"required,enum=active|inactive"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// When this payment term was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this payment term was last updated.
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
