package resourceregistry

import (
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeBatch,
		Load:       resourceloaders.LoadBatches,
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeBatchFlowNode,
		Load:       resourceloaders.LoadBatchFlowNodes,
	})
}
