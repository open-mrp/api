package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	recordstub.Register(constants.RecordTypeConversation, recordstub.FromResource(constants.ObjectTypeConversation,
		func(_ context.Context, c *apiresource.Conversation) *apiresource.Record {
			rec := apiresource.NewRecord(c.ID, constants.RecordTypeConversation)
			rec.Status = new(string(c.Status))
			rec.Metadata = map[string]string{}
			return rec
		}))
}
