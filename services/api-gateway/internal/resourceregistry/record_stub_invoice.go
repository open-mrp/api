package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	recordstub.Register(constants.RecordTypeInvoice, recordstub.FromResource(constants.ObjectTypeInvoice,
		func(_ context.Context, inv *apiresource.Invoice) *apiresource.Record {
			rec := apiresource.NewRecord(inv.ID, constants.RecordTypeInvoice)
			rec.Number = new(inv.Number)
			rec.Status = new(string(inv.PaymentStatus))
			return rec
		}))
}
