package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	recordstub.Register(constants.RecordTypeManufacturingMethod, recordstub.FromResource(constants.ObjectTypeManufacturingMethod,
		func(_ context.Context, m *apiresource.ManufacturingMethod) *apiresource.Record {
			rec := apiresource.NewRecord(m.ID, constants.RecordTypeManufacturingMethod)
			rec.Status = new(string(m.Status))
			return rec
		}))
}
