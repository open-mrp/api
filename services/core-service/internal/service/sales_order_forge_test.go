package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

func TestCancelOutcome(t *testing.T) {
	t.Parallel()

	assert.Equal(t, constants.SalesOrderStatusCodeCanceled, cancelOutcome(&domain.SalesOrder{}), "nothing shipped")
	shipped := time.Now()
	assert.Equal(t, constants.SalesOrderStatusCodeFulfilled, cancelOutcome(&domain.SalesOrder{FirstShipAt: &shipped}), "something shipped")
}

func TestSameSnapshot(t *testing.T) {
	t.Parallel()

	a := &domain.AddressSnapshot{Name: "Dock", Country: "US", City: new("Columbus")}
	assert.True(t, sameSnapshot(a, a.Clone()))
	b := a.Clone()
	b.City = new("Dayton")
	assert.False(t, sameSnapshot(a, b))
}

func TestShippingAddressFromSnapshot(t *testing.T) {
	t.Parallel()

	got := shippingAddressFromSnapshot(&domain.AddressSnapshot{Name: "Dock", Line1: new("1 Main"), State: new("OH"), PostalCode: new("43204"), Country: "US"})
	assert.Equal(t, "Dock", got.Name)
	assert.Equal(t, "1 Main", got.Street1)
	assert.Equal(t, "43204", got.Zip)
	assert.Equal(t, "US", got.Country)
	assert.Equal(t, domain.ShippingAddress{}, shippingAddressFromSnapshot(nil))
}
