package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

// messagingRecordSub is the include sub-field for a messaging field naming one record of any type, whose reference the presenter stashed as a recordstub.Ref under metaKey. A record of a type that has no stub loader yet is still shown by its type and ID, since that is all the conversation stores; a record that no longer exists is left null.
func messagingRecordSub(key string, ot constants.ObjectType, parentID func(parent any) string, metaKey string, set func(parent any, r *apiresource.Record)) include.SubField {
	ref := func(ctx context.Context, parent any) (recordstub.Ref, bool) {
		v, ok := include.GetLoadMeta(ctx).Get(ot, parentID(parent), metaKey)
		if !ok {
			return recordstub.Ref{}, false
		}
		r, ok := v.(recordstub.Ref)
		if !ok || !r.Type.IsValid() || r.ID == "" {
			return recordstub.Ref{}, false
		}
		return r, true
	}
	return include.SubField{
		Key:         key,
		Target:      constants.ObjectTypeRecord,
		Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			if r, ok := ref(ctx, parent); ok && recordstub.Registered(r.Type) {
				return []string{recordstub.Key(r.Type, r.ID)}
			}
			return nil
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			r, ok := ref(ctx, parent)
			if !ok {
				return
			}
			if !recordstub.Registered(r.Type) {
				stub := apiresource.NewRecord(r.ID, r.Type)
				stub.Metadata = map[string]string{}
				set(parent, stub)
				return
			}
			if s := recordstub.One(loaded, r.Type, r.ID); s != nil {
				set(parent, s)
			}
		},
	}
}
