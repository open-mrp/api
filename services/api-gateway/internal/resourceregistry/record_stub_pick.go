package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	recordstub.Register(constants.RecordTypePick, recordstub.FromResource(constants.ObjectTypePick,
		func(_ context.Context, p *apiresource.Pick) *apiresource.Record {
			rec := apiresource.NewRecord(p.ID, constants.RecordTypePick)
			rec.Number = new(p.Number)
			rec.Status = new(openClosedStatus(p.FinishedAt))
			return rec
		}))
}
