package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Whether the calling customer can currently contact support.
//
// `available` is true only when the vendor has configured a support route that resolves to at least one recipient. The customer portal gates its contact-support feature on this so customers never open a support thread no one is set up to receive.
type SupportAvailability struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=support_availability"`
	// Whether support can be contacted.
	Available bool `json:"available"`
}

var SampleSupportAvailability = &SupportAvailability{
	Object:    constants.ObjectTypeSupportAvailability,
	Available: true,
}

func (*SupportAvailability) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupportAvailability)
}
