package resourceloaders

import (
	"context"

	"github.com/open-mrp/apikit/include"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// StashAddressSnapshot is a document's snapshot as the response shows it, with the saved address it came from stashed
// for the `<key>.address` include (see resourceregistry's AddressSnapshotSub). key is the snapshot's field, such as
// `bill_to`. nil stays nil.
func StashAddressSnapshot(ctx context.Context, parentType constants.ObjectType, parentID, key string, p *pb.AddressSnapshot) *apiresource.AddressSnapshot {
	snap, addressID := AddressSnapshotFromProto(p)
	if addressID != "" {
		include.GetLoadMeta(ctx).Set(parentType, parentID, AddressSnapshotMetaKey(key), addressID)
	}
	return snap
}

// AddressSnapshotMetaKey is the load-meta key a snapshot's source address ID is stashed under.
func AddressSnapshotMetaKey(key string) string {
	return key + ".address_id"
}

// AddressSnapshotFromProto is a document's snapshot as the response shows it, and the ID of the saved address it came from ("" for inline) for the include that expands `address`. nil stays nil.
func AddressSnapshotFromProto(p *pb.AddressSnapshot) (*apiresource.AddressSnapshot, string) {
	if p == nil {
		return nil, ""
	}
	return &apiresource.AddressSnapshot{
		Name:       p.GetName(),
		Line1:      p.Line_1,
		Line2:      p.Line_2,
		City:       p.City,
		State:      p.State,
		PostalCode: p.PostalCode,
		Country:    p.GetCountry(),
		Phone:      p.Phone,
		Email:      p.Email,
	}, p.GetAddressId()
}
