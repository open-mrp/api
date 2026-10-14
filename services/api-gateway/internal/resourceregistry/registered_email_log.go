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
		ObjectType: constants.ObjectTypeEmailLog,
		Load:       resourceloaders.LoadEmailLogs,
		Subs: []include.SubField{
			{Key: "sent_by", Populate: populateSentByOnEmailLog},
		},
	})
}

func populateSentByOnEmailLog(ctx context.Context, parent any, _ map[string]any) {
	el := parent.(*apiresource.EmailLog)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeEmailLog, el.ID, "sent_by")
	if !ok {
		return
	}
	el.SentBy = v.(*apiresource.Actor)
}
