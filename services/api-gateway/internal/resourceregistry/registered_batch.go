package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeBatch,
		Load:       resourceloaders.LoadBatches,
		Subs: []resourcekit.SubField{
			// The measures are already on the batch — these exist so a caller can reach through them
			// to the unit each is counted in.
			{Key: "quantity", Target: constants.ObjectTypeQuantity, Cardinality: resourcekit.CardinalityOnePtr, ExtractRefs: extractQuantityRefFromBatch},
			{Key: "seconds", Target: constants.ObjectTypeQuantity, Cardinality: resourcekit.CardinalityOnePtr, ExtractRefs: extractSecondsRefFromBatch},
			{Key: "waste", Target: constants.ObjectTypeQuantity, Cardinality: resourcekit.CardinalityOnePtr, ExtractRefs: extractWasteRefFromBatch},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeBatchFlowNode,
		Load:       resourceloaders.LoadBatchFlowNodes,
		Subs: []resourcekit.SubField{
			{Key: "batch", Target: constants.ObjectTypeBatch, Cardinality: resourcekit.CardinalityOnePtr, ExtractRefs: extractBatchRefFromFlowNode},
		},
	})
}

// A flow node's batch is a value on the node, so the node reaches its measures through a pointer into
// that field rather than through a fetch: the batch is already whole, only its units are not.
func extractBatchRefFromFlowNode(_ context.Context, parent any) []any {
	return []any{&parent.(*apiresource.BatchFlowNode).Batch}
}

func extractQuantityRefFromBatch(_ context.Context, parent any) []any {
	return batchMeasureRef(parent.(*apiresource.Batch).Quantity)
}

func extractSecondsRefFromBatch(_ context.Context, parent any) []any {
	return batchMeasureRef(parent.(*apiresource.Batch).Seconds)
}

func extractWasteRefFromBatch(_ context.Context, parent any) []any {
	return batchMeasureRef(parent.(*apiresource.Batch).Waste)
}

func batchMeasureRef(q *apiresource.Quantity) []any {
	if q == nil {
		return nil
	}
	return []any{q}
}
