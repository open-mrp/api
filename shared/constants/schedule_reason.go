package constants

// ScheduleChangeReason explains why a plan was changed by hand.
//
// Distinct from ScheduleDeviationType, which names *what* changed about a line. The type is derived from the change itself; the reason is what the person supplies, and only a change inside a frozen week is required to supply one.
type ScheduleChangeReason string

const (
	// ScheduleChangeReasonMachineDown indicates the machine the campaign was on stopped running.
	ScheduleChangeReasonMachineDown ScheduleChangeReason = "machine_down"
	// ScheduleChangeReasonMaterialShortage indicates the material the campaign needs did not arrive.
	ScheduleChangeReasonMaterialShortage ScheduleChangeReason = "material_shortage"
	// ScheduleChangeReasonRushOrder indicates demand that could not wait for the next plan.
	ScheduleChangeReasonRushOrder ScheduleChangeReason = "rush_order"
	// ScheduleChangeReasonQualityHold indicates the work was stopped for a quality problem.
	ScheduleChangeReasonQualityHold ScheduleChangeReason = "quality_hold"
	// ScheduleChangeReasonOverRun indicates the floor produced more than the plan asked for.
	ScheduleChangeReasonOverRun ScheduleChangeReason = "over_run"
	// ScheduleChangeReasonUnderRun indicates the floor produced less than the plan asked for.
	ScheduleChangeReasonUnderRun ScheduleChangeReason = "under_run"
	// ScheduleChangeReasonCapacityChange indicates the available machine time changed, such as a shutdown or an added shift.
	ScheduleChangeReasonCapacityChange ScheduleChangeReason = "capacity_change"
	// ScheduleChangeReasonOther indicates a reason outside the list, which should be explained in the note.
	ScheduleChangeReasonOther ScheduleChangeReason = "other"
)

func (r ScheduleChangeReason) IsValid() bool {
	switch r {
	case ScheduleChangeReasonMachineDown, ScheduleChangeReasonMaterialShortage,
		ScheduleChangeReasonRushOrder, ScheduleChangeReasonQualityHold,
		ScheduleChangeReasonOverRun, ScheduleChangeReasonUnderRun,
		ScheduleChangeReasonCapacityChange, ScheduleChangeReasonOther:
		return true
	default:
		return false
	}
}

func (r ScheduleChangeReason) EnumValues() []string {
	return []string{
		string(ScheduleChangeReasonMachineDown),
		string(ScheduleChangeReasonMaterialShortage),
		string(ScheduleChangeReasonRushOrder),
		string(ScheduleChangeReasonQualityHold),
		string(ScheduleChangeReasonOverRun),
		string(ScheduleChangeReasonUnderRun),
		string(ScheduleChangeReasonCapacityChange),
		string(ScheduleChangeReasonOther),
	}
}

func (r *ScheduleChangeReason) StringPtr() *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}

// ScheduleChangeReasonPtr converts a stored string into the typed reason, returning nil when there is nothing recorded.
//
// Unknown values are returned as-is rather than dropped: a reason written before a code was retired is still the honest answer to why a plan changed, and silently blanking it would make the deviation log lie.
func ScheduleChangeReasonPtr(value *string) *ScheduleChangeReason {
	if value == nil {
		return nil
	}
	reason := ScheduleChangeReason(*value)
	return &reason
}

// DemandOverrideReason explains why the demand a plan is solved against was adjusted by hand.
type DemandOverrideReason string

const (
	// DemandOverrideReasonNewCustomer indicates demand from a customer with no order history.
	DemandOverrideReasonNewCustomer DemandOverrideReason = "new_customer"
	// DemandOverrideReasonLostAccount indicates demand that history contains but the future will not.
	DemandOverrideReasonLostAccount DemandOverrideReason = "lost_account"
	// DemandOverrideReasonPromotion indicates a planned campaign that history cannot predict.
	DemandOverrideReasonPromotion DemandOverrideReason = "promotion"
	// DemandOverrideReasonSeasonalShift indicates a season arriving earlier or later than usual.
	DemandOverrideReasonSeasonalShift DemandOverrideReason = "seasonal_shift"
	// DemandOverrideReasonNewProduct indicates an item with no history to forecast from.
	DemandOverrideReasonNewProduct DemandOverrideReason = "new_product"
	// DemandOverrideReasonDiscontinued indicates an item being wound down.
	DemandOverrideReasonDiscontinued DemandOverrideReason = "discontinued"
	// DemandOverrideReasonMarketIntelligence indicates knowledge of the market that the order book does not yet show.
	DemandOverrideReasonMarketIntelligence DemandOverrideReason = "market_intelligence"
	// DemandOverrideReasonOther indicates a reason outside the list, which should be explained in the note.
	DemandOverrideReasonOther DemandOverrideReason = "other"
)

func (r DemandOverrideReason) IsValid() bool {
	switch r {
	case DemandOverrideReasonNewCustomer, DemandOverrideReasonLostAccount,
		DemandOverrideReasonPromotion, DemandOverrideReasonSeasonalShift,
		DemandOverrideReasonNewProduct, DemandOverrideReasonDiscontinued,
		DemandOverrideReasonMarketIntelligence, DemandOverrideReasonOther:
		return true
	default:
		return false
	}
}

func (r DemandOverrideReason) EnumValues() []string {
	return []string{
		string(DemandOverrideReasonNewCustomer),
		string(DemandOverrideReasonLostAccount),
		string(DemandOverrideReasonPromotion),
		string(DemandOverrideReasonSeasonalShift),
		string(DemandOverrideReasonNewProduct),
		string(DemandOverrideReasonDiscontinued),
		string(DemandOverrideReasonMarketIntelligence),
		string(DemandOverrideReasonOther),
	}
}

func (r *DemandOverrideReason) StringPtr() *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}

// DemandOverrideReasonPtr converts a stored string into the typed reason, returning nil when there is nothing recorded.
func DemandOverrideReasonPtr(value *string) *DemandOverrideReason {
	if value == nil {
		return nil
	}
	reason := DemandOverrideReason(*value)
	return &reason
}

// MachineDowntimeReasonCode identifies why a machine stopped.
//
// The set matches the seeded `machine_downtime_reason` taxonomy, whose rows carry the OEE bucket each reason charges. Merchant-defined reasons are a later phase; when they land this stops being a closed set.
type MachineDowntimeReasonCode string

const (
	// MachineDowntimeReasonCodeBreakdown indicates an unplanned mechanical or electrical failure.
	MachineDowntimeReasonCodeBreakdown MachineDowntimeReasonCode = "breakdown"
	// MachineDowntimeReasonCodeChangeover indicates a yarn or style change.
	MachineDowntimeReasonCodeChangeover MachineDowntimeReasonCode = "changeover"
	// MachineDowntimeReasonCodeMaterialShortage indicates the machine had nothing to run.
	MachineDowntimeReasonCodeMaterialShortage MachineDowntimeReasonCode = "material_shortage"
	// MachineDowntimeReasonCodeNoOperator indicates the machine was staffed by nobody.
	MachineDowntimeReasonCodeNoOperator MachineDowntimeReasonCode = "no_operator"
	// MachineDowntimeReasonCodePlannedMaintenance indicates preventive maintenance scheduled in advance.
	MachineDowntimeReasonCodePlannedMaintenance MachineDowntimeReasonCode = "planned_maintenance"
	// MachineDowntimeReasonCodeMinorStop indicates a short stoppage that costs speed rather than availability.
	MachineDowntimeReasonCodeMinorStop MachineDowntimeReasonCode = "minor_stop"
	// MachineDowntimeReasonCodeQualityHold indicates the machine was stopped over a quality problem.
	MachineDowntimeReasonCodeQualityHold MachineDowntimeReasonCode = "quality_hold"
	// MachineDowntimeReasonCodeNoSchedule indicates time the machine was never planned to run, which is removed from the OEE calculation rather than counted against it.
	MachineDowntimeReasonCodeNoSchedule MachineDowntimeReasonCode = "no_schedule"
)

func (r MachineDowntimeReasonCode) IsValid() bool {
	switch r {
	case MachineDowntimeReasonCodeBreakdown, MachineDowntimeReasonCodeChangeover,
		MachineDowntimeReasonCodeMaterialShortage, MachineDowntimeReasonCodeNoOperator,
		MachineDowntimeReasonCodePlannedMaintenance, MachineDowntimeReasonCodeMinorStop,
		MachineDowntimeReasonCodeQualityHold, MachineDowntimeReasonCodeNoSchedule:
		return true
	default:
		return false
	}
}

func (r MachineDowntimeReasonCode) EnumValues() []string {
	return []string{
		string(MachineDowntimeReasonCodeBreakdown),
		string(MachineDowntimeReasonCodeChangeover),
		string(MachineDowntimeReasonCodeMaterialShortage),
		string(MachineDowntimeReasonCodeNoOperator),
		string(MachineDowntimeReasonCodePlannedMaintenance),
		string(MachineDowntimeReasonCodeMinorStop),
		string(MachineDowntimeReasonCodeQualityHold),
		string(MachineDowntimeReasonCodeNoSchedule),
	}
}

func (r *MachineDowntimeReasonCode) StringPtr() *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}
