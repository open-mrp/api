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
		ObjectType: constants.ObjectTypeAnnouncement,
		Load:       resourceloaders.LoadAnnouncements,
		Subs: []include.SubField{
			messagingRecordSub("related.record", constants.ObjectTypeAnnouncement,
				func(p any) string { return p.(*apiresource.Announcement).ID }, chatmap.MetaRelatedRecord,
				func(p any, r *apiresource.Record) { p.(*apiresource.Announcement).Related.Record = r }),
		},
	})
}
