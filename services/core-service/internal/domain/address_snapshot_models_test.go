package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func TestSnapshotAddress(t *testing.T) {
	str := func(s string) *string { return &s }
	addr := &domain.Address{
		ID: "ad_saved", Name: "Main warehouse", Phone: str("+15555550100"), Email: str("dock@example.com"),
		Geolocation: &domain.Geolocation{StreetLine1: str("1 Dock Rd"), StreetLine2: str("Bay 4"), Locality: str("Springfield"),
			State: str("IL"), PostalCode: str("62701"), Country: "US"},
	}

	got := domain.SnapshotAddress(addr)
	assert.Equal(t, &domain.AddressSnapshot{
		AddressID: str("ad_saved"), Name: "Main warehouse", Line1: str("1 Dock Rd"), Line2: str("Bay 4"), City: str("Springfield"),
		State: str("IL"), PostalCode: str("62701"), Country: "US", Phone: str("+15555550100"), Email: str("dock@example.com"),
	}, got)

	*addr.Geolocation.StreetLine1 = "2 Other St"
	addr.Name = "Renamed"
	assert.Equal(t, "1 Dock Rd", *got.Line1, "editing the address leaves the snapshot alone")
	assert.Equal(t, "Main warehouse", got.Name)

	assert.Nil(t, domain.SnapshotAddress(nil))
}
