package resourceregistry

import (
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeEmailLog,
		Load:       resourceloaders.LoadEmailLogs,
		Subs: []include.SubField{
			messagingRecordSub("related.record", constants.ObjectTypeEmailLog,
				func(p any) string { return p.(*apiresource.EmailLog).ID }, chatmap.MetaRelatedRecord,
				func(p any, r *apiresource.Record) { p.(*apiresource.EmailLog).Related.Record = r }),
		},
	})
}
