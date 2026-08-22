package grpc

import (
	"github.com/open-mrp/api/services/core-service/internal/domain"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// This file maps the shared fuzzy reference protobuf messages to their domain forms.
// A nil message yields the zero identifier (all fields empty), which resolvers treat as
// unset; the *Ptr variants return nil for an omitted optional reference so callers can
// tell "not provided" from an empty one.

func unitIdentifierFromProto(r *pb.UnitIdentifier) domain.UnitIdentifier {
	if r == nil {
		return domain.UnitIdentifier{}
	}
	return domain.UnitIdentifier{
		ID:           r.GetId(),
		Name:         r.GetName(),
		Abbreviation: r.GetAbbreviation(),
	}
}

func unitIdentifierPtrFromProto(r *pb.UnitIdentifier) *domain.UnitIdentifier {
	if r == nil {
		return nil
	}
	identifier := unitIdentifierFromProto(r)
	return &identifier
}

func itemIdentifierFromProto(r *pb.ItemIdentifier) domain.ItemIdentifier {
	if r == nil {
		return domain.ItemIdentifier{}
	}
	return domain.ItemIdentifier{
		ID:  r.GetId(),
		SKU: r.GetSku(),
	}
}

func objectIdentifierFromProto(r *pb.ObjectIdentifier) domain.ObjectIdentifier {
	if r == nil {
		return domain.ObjectIdentifier{}
	}
	return domain.ObjectIdentifier{
		ID:   r.GetId(),
		Name: r.GetName(),
	}
}

func objectIdentifierPtrFromProto(r *pb.ObjectIdentifier) *domain.ObjectIdentifier {
	if r == nil {
		return nil
	}
	identifier := objectIdentifierFromProto(r)
	return &identifier
}
