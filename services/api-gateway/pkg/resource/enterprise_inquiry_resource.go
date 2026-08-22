package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleEnterpriseInquiryID = "enir_w61ojhgj9sna"

// A submitted request to upgrade to an enterprise plan, routed to the sales team for follow-up.
type EnterpriseInquiry struct {
	// Enterprise inquiry ID.
	//
	// Inquiries are handed off to the sales team rather than stored as a queryable resource, so this identifier is only a reference to the submission you just made.
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
