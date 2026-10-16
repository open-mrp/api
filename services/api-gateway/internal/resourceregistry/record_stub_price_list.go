package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	recordstub.Register(constants.RecordTypePriceList, recordstub.FromResource(constants.ObjectTypePriceList,
		func(_ context.Context, l *apiresource.PriceList) *apiresource.Record {
			rec := apiresource.NewRecord(l.ID, constants.RecordTypePriceList)
			rec.Status = new(string(l.Status))
			return rec
		}))
}
