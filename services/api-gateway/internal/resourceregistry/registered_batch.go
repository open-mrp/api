package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeBatch,
		Load:       resourceloaders.LoadBatches,
		Subs: []include.SubField{
			// The measures are already on the batch — these exist so a caller can reach through them
			// to the unit each is counted in.
			{Key: "quantity", Target: constants.ObjectTypeQuantity, Cardinality: include.CardinalityOnePtr, ExtractRefs: extractQuantityRefFromBatch},
			{Key: "seconds", Target: constants.ObjectTypeQuantity, Cardinality: include.CardinalityOnePtr, ExtractRefs: extractSecondsRefFromBatch},
			{Key: "waste", Target: constants.ObjectTypeQuantity, Cardinality: include.CardinalityOnePtr, ExtractRefs: extractWasteRefFromBatch},
		},
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeBatchFlowNode,
		Load:       resourceloaders.LoadBatchFlowNodes,
		Subs: []include.SubField{
			{Key: "batch", Target: constants.ObjectTypeBatch, Cardinality: include.CardinalityOnePtr, ExtractRefs: extractBatchRefFromFlowNode},
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
