package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleEnterpriseInquiryID = "enir_01gf7a8200eaj8fke1xvw4h50x"

// EnterpriseInquiry represents a request for an enterprise plan upgrade.
type EnterpriseInquiry struct {
	// The unique identifier for this enterprise inquiry.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
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
