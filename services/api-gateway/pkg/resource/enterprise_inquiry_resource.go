package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

const SampleEnterpriseInquiryID = "eniq_01gf7a8200eaj8fke1xvw4h50x"

type EnterpriseInquiry struct {
	ID        string               `json:"id" validate:"required"`
	Object    constants.ObjectType `json:"object" validate:"required,enum=enterprise_inquiry"`
	CreatedAt time.Time            `json:"created_at" validate:"required"`
}

func (*EnterpriseInquiry) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEnterpriseInquiry)
}

var SampleEnterpriseInquiry = &EnterpriseInquiry{
	ID:        SampleEnterpriseInquiryID,
	Object:    constants.ObjectTypeEnterpriseInquiry,
	CreatedAt: time.Now(),
}
