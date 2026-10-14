package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	recordstub.Register(constants.RecordTypeSalesOrder, recordstub.FromResource(constants.ObjectTypeSalesOrder,
		func(_ context.Context, so *apiresource.SalesOrder) *apiresource.Record {
			rec := apiresource.NewRecord(so.ID, constants.RecordTypeSalesOrder)
			rec.Number = new(so.Number)
			rec.Status = new(string(so.Status))
			return rec
		}))
}
