package grpc

import (
	"github.com/open-mrp/api/services/core-service/internal/domain"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// AddressSnapshotToProto is a document's address snapshot as the RPC carries it; nil stays nil.
func AddressSnapshotToProto(s *domain.AddressSnapshot) *pb.AddressSnapshot {
	if s == nil {
		return nil
	}
	return &pb.AddressSnapshot{
		AddressId:  s.AddressID,
		Name:       s.Name,
		Line_1:     s.Line1,
		Line_2:     s.Line2,
		City:       s.City,
		State:      s.State,
		PostalCode: s.PostalCode,
		Country:    s.Country,
		Phone:      s.Phone,
		Email:      s.Email,
	}
}

// AddressSnapshotFromProto is the RPC's address snapshot; nil stays nil.
func AddressSnapshotFromProto(p *pb.AddressSnapshot) *domain.AddressSnapshot {
	if p == nil {
		return nil
	}
	return &domain.AddressSnapshot{
		AddressID:  p.AddressId,
		Name:       p.Name,
		Line1:      p.Line_1,
		Line2:      p.Line_2,
		City:       p.City,
		State:      p.State,
		PostalCode: p.PostalCode,
		Country:    p.Country,
		Phone:      p.Phone,
		Email:      p.Email,
	}
}
