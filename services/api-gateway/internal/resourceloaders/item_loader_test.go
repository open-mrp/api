package resourceloaders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func TestItemFromProto_UnsoldItemCarriesItsPriceOnlyForPreview9(t *testing.T) {
	t.Parallel()

	material := &pb.ItemInfo{Id: "itm_1", ItemTypeCode: "material", UnitValue: &pb.UnitPrice{Amount: "4.25", Currency: "usd", PerUnitId: "each"}}

	assert.Nil(t, ItemFromProto(material, false).Sales, "forge: a material without a product row has no sales section")

	carried := ItemFromProto(material, true).Sales
	require.NotNil(t, carried)
	require.NotNil(t, carried.UnitPrice)
	assert.Equal(t, "4.25", carried.UnitPrice.Amount)
	assert.Equal(t, constants.CustomerPortalVisibilityHidden, carried.PortalVisibility)

	zero := &pb.ItemInfo{Id: "itm_3", ItemTypeCode: "part", UnitValue: &pb.UnitPrice{Amount: "0.000000", Currency: "usd", PerUnitId: "each"}}
	assert.Nil(t, ItemFromProto(zero, true).Sales, "preview.9 reads a zero price as no unit_value")
}

func TestItemFromProto_SoldItemKeepsItsSection(t *testing.T) {
	t.Parallel()

	price := &pb.UnitPrice{Amount: "9.00", Currency: "usd", PerUnitId: "each"}
	product := &pb.ItemInfo{
		Id: "itm_2", ItemTypeCode: "product", UnitValue: price,
		Sales: &pb.ItemSalesInfo{UnitPrice: price, PortalVisibility: "visible"},
	}
	for _, preview9 := range []bool{false, true} {
		sales := ItemFromProto(product, preview9).Sales
		require.NotNil(t, sales)
		assert.Equal(t, constants.CustomerPortalVisibilityVisible, sales.PortalVisibility)
	}
}
