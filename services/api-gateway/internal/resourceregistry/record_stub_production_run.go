package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	recordstub.Register(constants.RecordTypeProductionRun, recordstub.FromResource(constants.ObjectTypeProductionRun,
		func(_ context.Context, pr *apiresource.ProductionRun) *apiresource.Record {
			rec := apiresource.NewRecord(pr.ID, constants.RecordTypeProductionRun)
			rec.Number = new(pr.Number)
			rec.Status = new(openClosedStatus(pr.CompletedAt))
			return rec
		}))
}
