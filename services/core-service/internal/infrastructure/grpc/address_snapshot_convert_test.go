package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func TestAddressSnapshotProto_RoundTrips(t *testing.T) {
	str := func(s string) *string { return &s }
	for _, s := range []*domain.AddressSnapshot{
		{AddressID: str("ad_saved"), Name: "Main warehouse", Line1: str("1 Dock Rd"), Line2: str("Bay 4"), City: str("Springfield"),
			State: str("IL"), PostalCode: str("62701"), Country: "US", Phone: str("+15555550100"), Email: str("dock@example.com")},
		{Name: "Inline", Country: "CA"},
		nil,
	} {
		assert.Equal(t, s, AddressSnapshotFromProto(AddressSnapshotToProto(s)))
	}
}
