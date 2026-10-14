package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

func TestAddressSnapshotColumns_RoundTrip(t *testing.T) {
	str := func(s string) *string { return &s }
	for _, s := range []*domain.AddressSnapshot{
		{AddressID: str("ad_saved"), Name: "Main warehouse", Line1: str("1 Dock Rd"), City: str("Springfield"), State: str("IL"),
			PostalCode: str("62701"), Country: "US", Email: str("dock@example.com")},
		{Name: "Inline", Country: "CA"},
	} {
		assert.Equal(t, s, AddressSnapshotToColumns(s).Snapshot())
	}

	none := AddressSnapshotToColumns(nil)
	assert.Equal(t, AddressSnapshotColumns{}, none)
	assert.Nil(t, none.Snapshot(), "no country means no snapshot")
}

func TestNewAddressSnapshotBackfill_BuildsOneUpdate(t *testing.T) {
	q := sqlc.New(nil)
	b, err := NewAddressSnapshotBackfill(q, q, "sales_order", "billing_address_id", "bill_to")
	require.NoError(t, err)
	assert.Equal(t, "UPDATE `sales_order` d JOIN address a ON a.id = d.`billing_address_id` JOIN geolocation g ON g.id = a.geolocation_id SET "+
		"d.`bill_to_address_id` = a.id, d.`bill_to_name` = a.name, d.`bill_to_line_1` = g.street_line_1, d.`bill_to_line_2` = g.street_line_2, "+
		"d.`bill_to_city` = g.locality, d.`bill_to_state` = g.state, d.`bill_to_postal_code` = g.postal_code, d.`bill_to_country` = g.country, "+
		"d.`bill_to_phone` = a.phone, d.`bill_to_email` = a.email WHERE d.`bill_to_country` IS NULL AND d.id IN ", b.update)

	_, err = NewAddressSnapshotBackfill(q, q, "sales_order", "billing_address_id; --", "bill_to")
	assert.Error(t, err)
}
