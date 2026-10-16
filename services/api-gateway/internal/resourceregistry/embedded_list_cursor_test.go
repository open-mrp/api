package resourceregistry

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func subField(t *testing.T, ot constants.ObjectType, key string) include.SubField {
	t.Helper()
	def := include.Lookup(ot)
	require.NotNil(t, def)
	for _, sub := range def.Subs {
		if sub.Key == key {
			return sub
		}
	}
	t.Fatalf("%s has no %q include", ot, key)
	return include.SubField{}
}

func TestWorkCenterMembers_NextPageContinuesTheFilteredList(t *testing.T) {
	t.Parallel()
	ctx := include.WithLoadMeta(context.Background())
	meta := include.GetLoadMeta(ctx)

	loaded := map[string]any{}
	var machineIDs []string
	for i := range 11 {
		id := fmt.Sprintf("mach_%02d", i)
		machineIDs = append(machineIDs, id)
		loaded[id] = &apiresource.Machine{ID: id, Object: constants.ObjectTypeMachine}
	}
	meta.Set(constants.ObjectTypeWorkCenter, "wkcc_1", "machine_ids", machineIDs)
	meta.Set(constants.ObjectTypeWorkCenter, "wkcc_1", resourceloaders.WorkCenterMachinesCursorKey, "cur+1")
	meta.Set(constants.ObjectTypeWorkCenter, "wkcc_1", "scanning_station_ids", []string{})
	meta.Set(constants.ObjectTypeWorkCenter, "wkcc_1", resourceloaders.WorkCenterStationsCursorKey, "")

	wc := &apiresource.WorkCenter{ID: "wkcc_1"}
	subField(t, constants.ObjectTypeWorkCenter, "machines").Populate(ctx, wc, loaded)
	subField(t, constants.ObjectTypeWorkCenter, "scanning_stations").Populate(ctx, wc, loaded)

	require.NotNil(t, wc.Machines)
	assert.Len(t, wc.Machines.Data, apiresource.EmbeddedListLimit)
	require.NotNil(t, wc.Machines.PageInfo.NextPageURL)
	assert.Equal(t, "/v1/operations/machines?work_center_ids%5B%5D=wkcc_1&cursor=cur%2B1", *wc.Machines.PageInfo.NextPageURL)

	require.NotNil(t, wc.ScanningStations)
	assert.Empty(t, wc.ScanningStations.Data)
	assert.Nil(t, wc.ScanningStations.PageInfo.NextPageURL)
}

func TestLocationChildren_NextPageContinuesTheFilteredList(t *testing.T) {
	t.Parallel()
	ctx := include.WithLoadMeta(context.Background())
	meta := include.GetLoadMeta(ctx)

	loaded := map[string]any{}
	var childIDs []string
	for i := range 11 {
		id := fmt.Sprintf("loc_%02d", i)
		childIDs = append(childIDs, id)
		loaded[id] = &apiresource.Location{ID: id, Object: constants.ObjectTypeLocation}
	}
	meta.Set(constants.ObjectTypeLocation, "loc_parent", "child_ids", childIDs)
	meta.Set(constants.ObjectTypeLocation, "loc_parent", resourceloaders.LocationChildrenCursorKey, "cur")

	loc := &apiresource.Location{ID: "loc_parent"}
	subField(t, constants.ObjectTypeLocation, "children").Populate(ctx, loc, loaded)

	require.NotNil(t, loc.Children)
	assert.Len(t, loc.Children.Data, apiresource.EmbeddedListLimit)
	assert.Equal(t, "loc_09", loc.Children.Data[9].ID)
	require.NotNil(t, loc.Children.PageInfo.NextPageURL)
	assert.Equal(t, "/v1/operations/locations?parent_ids%5B%5D=loc_parent&cursor=cur", *loc.Children.PageInfo.NextPageURL)
}
