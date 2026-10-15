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
		ObjectType: constants.ObjectTypeMessageAttachment,
		Load:       resourceloaders.LoadMessageAttachments,
		Subs: []include.SubField{
			// The attachment is reached through its message's attachments include.
			messagingRecordSub("record", constants.ObjectTypeMessageAttachment,
				func(p any) string { return p.(*apiresource.MessageAttachment).ID }, chatmap.MetaRecord,
				func(p any, r *apiresource.Record) { p.(*apiresource.MessageAttachment).Record = r }),
		},
	})
}
