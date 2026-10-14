package resourceloaders

import (
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	pb "github.com/open-mrp/api/shared/proto/core"
)

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
