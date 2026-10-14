package resourceregistry

import (
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	"github.com/open-mrp/api/shared/constants"
)

// Record stubs of every type load through this one Definition: recordstub.Load sends each key to its type's loader, registered in record_stub_<type>.go.
func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeRecord,
		Load:       recordstub.Load,
	})
}
