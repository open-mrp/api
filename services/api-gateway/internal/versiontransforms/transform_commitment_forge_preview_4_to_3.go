package versiontransforms

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&commitmentForgePreview4To3{})
}

// commitmentForgePreview4To3 downgrades commitment-carrying payloads from 1.0.forge-preview.4 to 1.0.forge-preview.3.
//
// preview.4 gathered every ship-by field a sales order, a pick, and a commitment preview carried flat into one `commitment` sub-resource shared by all three — the inputs a caller writes (promised_at, lead_time_override_days, ship_by_override_date) alongside the date they resolve to and its derivation. preview.3 read them all off the parent, so the downgrade hoists them back and drops the sub-resource.
//
// It also splits ship_by_date back in two. preview.4 reports one instant — the ship-by day at the plant's pickup cutoff, or that day at midnight when no cutoff is set — where preview.3 carried the day in ship_by_date and the cutoff separately in ship_by_cutoff_at. Splitting on midnight UTC recovers both exactly whenever the cutoff falls on the same UTC day as the date it belongs to, which is every plant whose zone and cutoff do not straddle midnight in UTC. For the few that do, preview.3 sees the ship-by day the cutoff instant lands on rather than the one it was computed for; preview.4 is where that distinction is expressible.
//
// Only the keys each parent actually carried in preview.3 are restored. Writing the full set everywhere would invent fields for readers that never saw them.
type commitmentForgePreview4To3 struct{}

func (t *commitmentForgePreview4To3) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview4
}

func (t *commitmentForgePreview4To3) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview3
}

func (t *commitmentForgePreview4To3) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{
		constants.ObjectTypeSalesOrder,
		constants.ObjectTypePick,
		constants.ObjectTypeSalesOrderCommitmentQuote,
	}
}

func (t *commitmentForgePreview4To3) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	downgradeCommitmentsIn(data)
	return data
}

func (t *commitmentForgePreview4To3) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	// The commitment is derived and never submitted; the inputs that produce it stayed flat on the request in both versions.
	return data
}

// The keys each parent carried flat in preview.3. ship_by_date and ship_by_cutoff_at are written by the split below rather than copied, so they are absent here.
var preview3CommitmentKeys = map[string][]string{
	string(constants.ObjectTypePick): {
		"promised_at", "lead_time_days", "lead_time_source", "transit_days", "transit_source",
	},
	string(constants.ObjectTypeSalesOrder): {
		"promised_at", "lead_time_override_days", "ship_by_override_date",
		"calendar_adjustment_days", "lead_time_days", "lead_time_source",
		"transit_days", "transit_source",
	},
	string(constants.ObjectTypeSalesOrderCommitmentQuote): {
		"lead_time_days", "lead_time_source", "transit_days", "transit_source",
		"estimated_delivery_date", "calendar_adjustment_days",
	},
}

// Which parents carried the cutoff as its own key. A pick never did, so its ship-by downgrades to the bare day.
var preview3CutoffCarriers = map[string]bool{
	string(constants.ObjectTypeSalesOrder):                true,
	string(constants.ObjectTypeSalesOrderCommitmentQuote): true,
}

// downgradeCommitmentsIn walks the payload and rewrites every commitment-carrying object in place — a bare resource, a list envelope, and a pick or an order nested inside another resource.
func downgradeCommitmentsIn(node any) {
	switch v := node.(type) {
	case map[string]any:
		if keys, ok := preview3CommitmentKeys[asString(v["object"])]; ok {
			hoistCommitment(v, keys)
		}
		for _, child := range v {
			downgradeCommitmentsIn(child)
		}
	case []any:
		for _, child := range v {
			downgradeCommitmentsIn(child)
		}
	}
}

func hoistCommitment(parent map[string]any, keys []string) {
	// Reading from a nil map yields the nil the absent key downgrades to, so an object whose commitment was omitted still gets its preview.3 keys back as nulls rather than losing them.
	commitment, _ := parent["commitment"].(map[string]any)
	delete(parent, "commitment")

	for _, key := range keys {
		parent[key] = commitment[key]
	}

	object := asString(parent["object"])
	day, cutoff := splitShipBy(asString(commitment["ship_by_date"]))
	parent["ship_by_date"] = day
	if preview3CutoffCarriers[object] {
		parent["ship_by_cutoff_at"] = cutoff
	}

	// preview.3 reported the preview's calendar adjustment as a plain number, never null.
	if object == string(constants.ObjectTypeSalesOrderCommitmentQuote) && parent["calendar_adjustment_days"] == nil {
		parent["calendar_adjustment_days"] = 0
	}
}

// splitShipBy takes preview.4's single instant back to preview.3's day plus optional cutoff.
//
// Midnight UTC is what a ship-by with no cutoff serialized as in both versions, so it maps to the day alone; any other time is a real cutoff and is reported as one, with the day it lands on. An unparseable or absent value downgrades to two nulls rather than to a guess.
func splitShipBy(raw string) (any, any) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, nil
	}
	utc := t.UTC()
	day := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if utc.Equal(time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)) {
		return day, nil
	}
	return day, raw
}
