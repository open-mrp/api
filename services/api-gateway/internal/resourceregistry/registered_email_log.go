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
		ObjectType: constants.ObjectTypeEmailLog,
		Load:       resourceloaders.LoadEmailLogs,
		Subs: []resourcekit.SubField{
			{Key: "sent_by", Populate: populateSentByOnEmailLog},
		},
	})
}

func populateSentByOnEmailLog(ctx context.Context, parent any, _ map[string]any) {
	el := parent.(*apiresource.EmailLog)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeEmailLog, el.ID, "sent_by")
	if !ok {
		return
	}
	el.SentBy = v.(*apiresource.Actor)
}
