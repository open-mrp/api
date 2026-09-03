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
