package apiresource

import (
	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
)

const SampleManufacturingMethodID = "prmt_6gkh6rnutwgz"

// How an item is made: the operations it goes through and the materials it consumes.
type ManufacturingMethod struct {
	// Manufacturing method ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=manufacturing_method"`
}

var SampleManufacturingMethod = &ManufacturingMethod{
	ID:     SampleManufacturingMethodID,
	Object: constants.ObjectTypeManufacturingMethod,
}

func (*ManufacturingMethod) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleManufacturingMethod)
}
