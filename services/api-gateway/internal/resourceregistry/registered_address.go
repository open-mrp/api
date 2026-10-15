package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// Whoever builds an address stashes its receive calendar's ID (resourceloaders.StashReceiveCalendar); an address
	// built without one never shows a receive calendar.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAddress,
		Load:       resourceloaders.LoadAddresses,
		Subs: []include.SubField{
			{
				Key:         "receive_calendar",
				Target:      constants.ObjectTypeOperatingCalendar,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractReceiveCalendarIDFromAddress,
				Populate:    populateReceiveCalendarOnAddress,
			},
		},
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeOperatingCalendar,
		Load:       resourceloaders.LoadOperatingCalendars,
	})
}

func extractReceiveCalendarIDFromAddress(ctx context.Context, parent any) []string {
	a := parent.(*apiresource.Address)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeAddress, a.ID, "receive_calendar_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateReceiveCalendarOnAddress(ctx context.Context, parent any, loaded map[string]any) {
	a := parent.(*apiresource.Address)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeAddress, a.ID, "receive_calendar_id")
	if v, ok := loaded[id]; ok && id != "" {
		a.ReceiveCalendar = v.(*apiresource.OperatingCalendar)
	}
}
