package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	recordstub.Register(constants.RecordTypeShipment, recordstub.FromResource(constants.ObjectTypeShipment,
		func(ctx context.Context, s *apiresource.Shipment) *apiresource.Record {
			rec := apiresource.NewRecord(s.ID, constants.RecordTypeShipment)
			rec.Number = new(s.Number)
			rec.Status = new(string(s.Status))
			// The shipment loader stashes tracking, carrier and ship date for its stubs.
			if m, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "record_metadata"); ok {
				if meta, ok := m.(map[string]string); ok && len(meta) > 0 {
					rec.Metadata = meta
				}
			}
			return rec
		}))
}
