package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleEnterpriseInquiryID = "enir_01ec571c64ecd75aaf4662fcd4"

// A submitted request to upgrade to an enterprise plan, routed to the sales team for follow-up.
type EnterpriseInquiry struct {
	// Enterprise inquiry ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=enterprise_inquiry"`
	// When this inquiry was submitted.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleEnterpriseInquiry = &EnterpriseInquiry{
	ID:        SampleEnterpriseInquiryID,
	Object:    constants.ObjectTypeEnterpriseInquiry,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*EnterpriseInquiry) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEnterpriseInquiry)
}
