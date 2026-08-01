package constants

// ProductionScheduleStatus is the lifecycle state of a schedule version.
type ProductionScheduleStatus string

const (
	// ProductionScheduleStatusDraft indicates the version is still editable and commits to nothing.
	ProductionScheduleStatusDraft ProductionScheduleStatus = "draft"
	// ProductionScheduleStatusGenerating indicates the solver is still building the version.
	ProductionScheduleStatusGenerating ProductionScheduleStatus = "generating"
	// ProductionScheduleStatusPublished indicates the version is live and its frozen weeks are committed.
	ProductionScheduleStatusPublished ProductionScheduleStatus = "published"
	// ProductionScheduleStatusSuperseded indicates a later version replaced this one over the same horizon.
	ProductionScheduleStatusSuperseded ProductionScheduleStatus = "superseded"
	// ProductionScheduleStatusArchived indicates the version was retired without being replaced.
	ProductionScheduleStatusArchived ProductionScheduleStatus = "archived"
	// ProductionScheduleStatusFailed indicates the solver could not produce a plan.
	ProductionScheduleStatusFailed ProductionScheduleStatus = "failed"
)

func (s ProductionScheduleStatus) IsValid() bool {
	switch s {
	case ProductionScheduleStatusDraft, ProductionScheduleStatusGenerating, ProductionScheduleStatusPublished,
		ProductionScheduleStatusSuperseded, ProductionScheduleStatusArchived, ProductionScheduleStatusFailed:
		return true
	default:
		return false
	}
}

func (s ProductionScheduleStatus) EnumValues() []string {
	return []string{
		string(ProductionScheduleStatusDraft),
		string(ProductionScheduleStatusGenerating),
		string(ProductionScheduleStatusPublished),
		string(ProductionScheduleStatusSuperseded),
		string(ProductionScheduleStatusArchived),
		string(ProductionScheduleStatusFailed),
	}
}

// ProductionScheduleLineStatus is the progress of one planned campaign.
type ProductionScheduleLineStatus string

const (
	// ProductionScheduleLineStatusPlanned indicates the campaign has not been released to the floor.
	ProductionScheduleLineStatusPlanned ProductionScheduleLineStatus = "planned"
	// ProductionScheduleLineStatusReleased indicates the campaign has been released to the floor.
	ProductionScheduleLineStatusReleased ProductionScheduleLineStatus = "released"
	// ProductionScheduleLineStatusInProgress indicates the campaign is being run.
	ProductionScheduleLineStatusInProgress ProductionScheduleLineStatus = "in_progress"
	// ProductionScheduleLineStatusComplete indicates the campaign finished.
	ProductionScheduleLineStatusComplete ProductionScheduleLineStatus = "complete"
	// ProductionScheduleLineStatusCancelled indicates the campaign will not be run.
	ProductionScheduleLineStatusCancelled ProductionScheduleLineStatus = "cancelled"
)

func (s ProductionScheduleLineStatus) IsValid() bool {
	switch s {
	case ProductionScheduleLineStatusPlanned, ProductionScheduleLineStatusReleased,
		ProductionScheduleLineStatusInProgress, ProductionScheduleLineStatusComplete,
		ProductionScheduleLineStatusCancelled:
		return true
	default:
		return false
	}
}

func (s ProductionScheduleLineStatus) EnumValues() []string {
	return []string{
		string(ProductionScheduleLineStatusPlanned),
		string(ProductionScheduleLineStatusReleased),
		string(ProductionScheduleLineStatusInProgress),
		string(ProductionScheduleLineStatusComplete),
		string(ProductionScheduleLineStatusCancelled),
	}
}

// ScheduleLineSource records who put a campaign on the plan. A regenerate has to be able to tell hand-placed work from solver output.
type ScheduleLineSource string

const (
	// ScheduleLineSourceSolver indicates the campaign came from the scheduling solver.
	ScheduleLineSourceSolver ScheduleLineSource = "solver"
	// ScheduleLineSourceManual indicates the campaign was placed or edited by hand.
	ScheduleLineSourceManual ScheduleLineSource = "manual"
)

func (s ScheduleLineSource) IsValid() bool {
	switch s {
	case ScheduleLineSourceSolver, ScheduleLineSourceManual:
		return true
	default:
		return false
	}
}

func (s ScheduleLineSource) EnumValues() []string {
	return []string{string(ScheduleLineSourceSolver), string(ScheduleLineSourceManual)}
}

// ScheduleDemandBasis is how demand was derived for a plan.
type ScheduleDemandBasis string

const (
	// ScheduleDemandBasisTrailing12 indicates demand came from the trailing twelve months of orders.
	ScheduleDemandBasisTrailing12 ScheduleDemandBasis = "trailing_12"
	// ScheduleDemandBasisSeasonalEMA indicates demand came from a seasonal exponential moving average.
	ScheduleDemandBasisSeasonalEMA ScheduleDemandBasis = "seasonal_ema"
)

func (b ScheduleDemandBasis) IsValid() bool {
	switch b {
	case ScheduleDemandBasisTrailing12, ScheduleDemandBasisSeasonalEMA:
		return true
	default:
		return false
	}
}

func (b ScheduleDemandBasis) EnumValues() []string {
	return []string{string(ScheduleDemandBasisTrailing12), string(ScheduleDemandBasisSeasonalEMA)}
}

// ScheduleGenerationSource records what caused a version to be generated.
type ScheduleGenerationSource string

const (
	// ScheduleGenerationSourceManual indicates a person asked for the version.
	ScheduleGenerationSourceManual ScheduleGenerationSource = "manual"
	// ScheduleGenerationSourceScheduled indicates the generation cadence produced the version.
	ScheduleGenerationSourceScheduled ScheduleGenerationSource = "scheduled"
)

func (s ScheduleGenerationSource) IsValid() bool {
	switch s {
	case ScheduleGenerationSourceManual, ScheduleGenerationSourceScheduled:
		return true
	default:
		return false
	}
}

func (s ScheduleGenerationSource) EnumValues() []string {
	return []string{string(ScheduleGenerationSourceManual), string(ScheduleGenerationSourceScheduled)}
}

// ScheduleDeviationType names what changed about a campaign.
type ScheduleDeviationType string

const (
	// ScheduleDeviationTypeLineAdded indicates a campaign was added by hand.
	ScheduleDeviationTypeLineAdded ScheduleDeviationType = "line_added"
	// ScheduleDeviationTypeLineRemoved indicates a campaign was removed.
	ScheduleDeviationTypeLineRemoved ScheduleDeviationType = "line_removed"
	// ScheduleDeviationTypeQuantityChanged indicates a campaign's quantity changed.
	ScheduleDeviationTypeQuantityChanged ScheduleDeviationType = "quantity_changed"
	// ScheduleDeviationTypeMachineChanged indicates a campaign moved to another machine.
	ScheduleDeviationTypeMachineChanged ScheduleDeviationType = "machine_changed"
	// ScheduleDeviationTypeResequenced indicates a campaign's position within its week changed.
	ScheduleDeviationTypeResequenced ScheduleDeviationType = "resequenced"
	// ScheduleDeviationTypeWeekMoved indicates a campaign moved to another week.
	ScheduleDeviationTypeWeekMoved ScheduleDeviationType = "week_moved"
)

func (d ScheduleDeviationType) IsValid() bool {
	switch d {
	case ScheduleDeviationTypeLineAdded, ScheduleDeviationTypeLineRemoved,
		ScheduleDeviationTypeQuantityChanged, ScheduleDeviationTypeMachineChanged,
		ScheduleDeviationTypeResequenced, ScheduleDeviationTypeWeekMoved:
		return true
	default:
		return false
	}
}

func (d ScheduleDeviationType) EnumValues() []string {
	return []string{
		string(ScheduleDeviationTypeLineAdded),
		string(ScheduleDeviationTypeLineRemoved),
		string(ScheduleDeviationTypeQuantityChanged),
		string(ScheduleDeviationTypeMachineChanged),
		string(ScheduleDeviationTypeResequenced),
		string(ScheduleDeviationTypeWeekMoved),
	}
}

// ScheduleFreezeStatus says whether a campaign or a change sits inside the committed part of a horizon.
type ScheduleFreezeStatus string

const (
	// ScheduleFreezeStatusFrozen indicates the campaign is a commitment; changing it requires a reason.
	ScheduleFreezeStatusFrozen ScheduleFreezeStatus = "frozen"
	// ScheduleFreezeStatusFlexible indicates the campaign is still a plan and can be changed freely.
	ScheduleFreezeStatusFlexible ScheduleFreezeStatus = "flexible"
)

func (f ScheduleFreezeStatus) IsValid() bool {
	switch f {
	case ScheduleFreezeStatusFrozen, ScheduleFreezeStatusFlexible:
		return true
	default:
		return false
	}
}

func (f ScheduleFreezeStatus) EnumValues() []string {
	return []string{string(ScheduleFreezeStatusFrozen), string(ScheduleFreezeStatusFlexible)}
}

// FreezeStatusOf maps the stored flag onto the enum the API exposes.
func FreezeStatusOf(frozen bool) ScheduleFreezeStatus {
	if frozen {
		return ScheduleFreezeStatusFrozen
	}
	return ScheduleFreezeStatusFlexible
}

// ABCClass ranks a SKU by how much of the constraint it consumes.
type ABCClass string

const (
	// ABCClassA indicates a SKU that consumes the largest share of constraint capacity.
	ABCClassA ABCClass = "a"
	// ABCClassB indicates a SKU with moderate constraint consumption.
	ABCClassB ABCClass = "b"
	// ABCClassC indicates a SKU that consumes little constraint capacity.
	ABCClassC ABCClass = "c"
)

func (c ABCClass) IsValid() bool {
	switch c {
	case ABCClassA, ABCClassB, ABCClassC:
		return true
	default:
		return false
	}
}

func (c ABCClass) EnumValues() []string {
	return []string{string(ABCClassA), string(ABCClassB), string(ABCClassC)}
}

// AttainmentGroupBy is the dimension a schedule-attainment breakdown is grouped by.
type AttainmentGroupBy string

const (
	// AttainmentGroupByWeek groups attainment by horizon week.
	AttainmentGroupByWeek AttainmentGroupBy = "week"
	// AttainmentGroupByMachine groups attainment by machine.
	AttainmentGroupByMachine AttainmentGroupBy = "machine"
	// AttainmentGroupByDepartment groups attainment by department.
	AttainmentGroupByDepartment AttainmentGroupBy = "department"
	// AttainmentGroupByItem groups attainment by item.
	AttainmentGroupByItem AttainmentGroupBy = "item"
)

func (g AttainmentGroupBy) IsValid() bool {
	switch g {
	case AttainmentGroupByWeek, AttainmentGroupByMachine, AttainmentGroupByDepartment, AttainmentGroupByItem:
		return true
	default:
		return false
	}
}

func (g AttainmentGroupBy) EnumValues() []string {
	return []string{
		string(AttainmentGroupByWeek),
		string(AttainmentGroupByMachine),
		string(AttainmentGroupByDepartment),
		string(AttainmentGroupByItem),
	}
}

// AttainmentBaselineStatus says whether a measured period had a plan to be measured against at all. A period with no published version has no attainment, which is a different statement from missing the plan entirely.
type AttainmentBaselineStatus string

const (
	// AttainmentBaselineStatusMeasured indicates a published version covered the period.
	AttainmentBaselineStatusMeasured AttainmentBaselineStatus = "measured"
	// AttainmentBaselineStatusNoBaseline indicates nothing was published over the period.
	AttainmentBaselineStatusNoBaseline AttainmentBaselineStatus = "no_baseline"
)

func (s AttainmentBaselineStatus) IsValid() bool {
	switch s {
	case AttainmentBaselineStatusMeasured, AttainmentBaselineStatusNoBaseline:
		return true
	default:
		return false
	}
}

func (s AttainmentBaselineStatus) EnumValues() []string {
	return []string{string(AttainmentBaselineStatusMeasured), string(AttainmentBaselineStatusNoBaseline)}
}

// SchedulePolicyConstraint names a limit the solver hit while sizing a SKU's campaigns. Modelled as a list of named constraints rather than one boolean per limit so a new limit does not add another flag to every response.
type SchedulePolicyConstraint string

const (
	// SchedulePolicyConstraintEOQCapped indicates the economic order quantity was capped at the maximum weeks of supply.
	SchedulePolicyConstraintEOQCapped SchedulePolicyConstraint = "eoq_capped"
	// SchedulePolicyConstraintCapacityStarved indicates demand exceeded the capacity available to the SKU.
	SchedulePolicyConstraintCapacityStarved SchedulePolicyConstraint = "capacity_starved"
)

func (c SchedulePolicyConstraint) IsValid() bool {
	switch c {
	case SchedulePolicyConstraintEOQCapped, SchedulePolicyConstraintCapacityStarved:
		return true
	default:
		return false
	}
}

func (c SchedulePolicyConstraint) EnumValues() []string {
	return []string{
		string(SchedulePolicyConstraintEOQCapped),
		string(SchedulePolicyConstraintCapacityStarved),
	}
}

func (s *ProductionScheduleStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (s *ProductionScheduleLineStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (s *ScheduleLineSource) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (b *ScheduleDemandBasis) StringPtr() *string {
	if b == nil {
		return nil
	}
	str := string(*b)
	return &str
}

func (s *ScheduleGenerationSource) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (d *ScheduleDeviationType) StringPtr() *string {
	if d == nil {
		return nil
	}
	str := string(*d)
	return &str
}

func (f *ScheduleFreezeStatus) StringPtr() *string {
	if f == nil {
		return nil
	}
	str := string(*f)
	return &str
}

func (c *ABCClass) StringPtr() *string {
	if c == nil {
		return nil
	}
	str := string(*c)
	return &str
}

func (g *AttainmentGroupBy) StringPtr() *string {
	if g == nil {
		return nil
	}
	str := string(*g)
	return &str
}

func (s *AttainmentBaselineStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (c *SchedulePolicyConstraint) StringPtr() *string {
	if c == nil {
		return nil
	}
	str := string(*c)
	return &str
}

// SettingsStatus says whether a settings resource holds values the merchant saved or the defaults the solver would otherwise apply. Exposed so a settings page can show "using defaults" rather than implying someone chose these numbers.
type SettingsStatus string

const (
	// SettingsStatusStored indicates the values were saved by the merchant.
	SettingsStatusStored SettingsStatus = "stored"
	// SettingsStatusDefault indicates the values are the solver's defaults.
	SettingsStatusDefault SettingsStatus = "default"
)

func (s SettingsStatus) IsValid() bool {
	switch s {
	case SettingsStatusStored, SettingsStatusDefault:
		return true
	default:
		return false
	}
}

func (s SettingsStatus) EnumValues() []string {
	return []string{string(SettingsStatusStored), string(SettingsStatusDefault)}
}

func (s *SettingsStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

// SettingsStatusOf maps the stored flag onto the enum the API exposes.
func SettingsStatusOf(stored bool) SettingsStatus {
	if stored {
		return SettingsStatusStored
	}
	return SettingsStatusDefault
}

// ScheduleResourceScope is what a per-resource planning override attaches to.
type ScheduleResourceScope string

const (
	// ScheduleResourceScopeMachine overrides planning for one machine.
	ScheduleResourceScopeMachine ScheduleResourceScope = "machine"
	// ScheduleResourceScopeDepartment overrides planning for one department.
	ScheduleResourceScopeDepartment ScheduleResourceScope = "department"
	// ScheduleResourceScopeProductionStep overrides planning for one production step.
	ScheduleResourceScopeProductionStep ScheduleResourceScope = "production_step"
)

func (s ScheduleResourceScope) IsValid() bool {
	switch s {
	case ScheduleResourceScopeMachine, ScheduleResourceScopeDepartment, ScheduleResourceScopeProductionStep:
		return true
	default:
		return false
	}
}

func (s ScheduleResourceScope) EnumValues() []string {
	return []string{
		string(ScheduleResourceScopeMachine),
		string(ScheduleResourceScopeDepartment),
		string(ScheduleResourceScopeProductionStep),
	}
}

func (s *ScheduleResourceScope) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

// ParticipationStatus says whether a resource takes part in planning.
//
// Machines are selected by department — the room that sets the pace of the factory — so this does not opt one in. It takes one out: a machine down for a rebuild should not have campaigns planned onto it, and the absence of a setting has to mean "planned" or adding a machine to the department would quietly do nothing.
type ParticipationStatus string

const (
	// ParticipationStatusIncluded indicates the resource is planned, which is the default for anything in the constraint department.
	ParticipationStatusIncluded ParticipationStatus = "included"
	// ParticipationStatusExcluded indicates the resource is deliberately left out of planning.
	ParticipationStatusExcluded ParticipationStatus = "excluded"
)

func (s ParticipationStatus) IsValid() bool {
	switch s {
	case ParticipationStatusIncluded, ParticipationStatusExcluded:
		return true
	default:
		return false
	}
}

func (s ParticipationStatus) EnumValues() []string {
	return []string{
		string(ParticipationStatusIncluded),
		string(ParticipationStatusExcluded),
	}
}

// ParticipationStatusOf maps the stored exclusion flag onto the status.
func ParticipationStatusOf(isExcluded bool) ParticipationStatus {
	if isExcluded {
		return ParticipationStatusExcluded
	}
	return ParticipationStatusIncluded
}

func (s *ParticipationStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

// ScheduleMergeMode says what a regenerate does with the hand edits already on a draft.
//
// There is no "preserve frozen" mode because only a published version has frozen lines and a published version is never regenerated in place — it is superseded by publishing a newer one. Regenerating a commitment the floor is already working to would rewrite history rather than replan.
type ScheduleMergeMode string

const (
	// ScheduleMergeModePreserveManual keeps every hand-edited campaign and replaces the rest with the fresh solve.
	ScheduleMergeModePreserveManual ScheduleMergeMode = "preserve_manual"
	// ScheduleMergeModeReplaceAll discards every hand edit and takes the fresh solve whole.
	ScheduleMergeModeReplaceAll ScheduleMergeMode = "replace_all"
)

func (m ScheduleMergeMode) IsValid() bool {
	switch m {
	case ScheduleMergeModePreserveManual, ScheduleMergeModeReplaceAll:
		return true
	default:
		return false
	}
}

func (m ScheduleMergeMode) EnumValues() []string {
	return []string{
		string(ScheduleMergeModePreserveManual),
		string(ScheduleMergeModeReplaceAll),
	}
}

func (m *ScheduleMergeMode) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}

// ScheduleDiffChange is what a regenerate would do to one campaign.
type ScheduleDiffChange string

const (
	// ScheduleDiffChangeAdded indicates a campaign the fresh solve wants that the current plan does not have.
	ScheduleDiffChangeAdded ScheduleDiffChange = "added"
	// ScheduleDiffChangeRemoved indicates a campaign the current plan has that the fresh solve does not want.
	ScheduleDiffChangeRemoved ScheduleDiffChange = "removed"
	// ScheduleDiffChangeChanged indicates a campaign both have, in a different quantity.
	ScheduleDiffChangeChanged ScheduleDiffChange = "changed"
	// ScheduleDiffChangeUnchanged indicates a campaign both agree on.
	ScheduleDiffChangeUnchanged ScheduleDiffChange = "unchanged"
)

func (c ScheduleDiffChange) IsValid() bool {
	switch c {
	case ScheduleDiffChangeAdded, ScheduleDiffChangeRemoved, ScheduleDiffChangeChanged, ScheduleDiffChangeUnchanged:
		return true
	default:
		return false
	}
}

func (c ScheduleDiffChange) EnumValues() []string {
	return []string{
		string(ScheduleDiffChangeAdded),
		string(ScheduleDiffChangeRemoved),
		string(ScheduleDiffChangeChanged),
		string(ScheduleDiffChangeUnchanged),
	}
}

func (c *ScheduleDiffChange) StringPtr() *string {
	if c == nil {
		return nil
	}
	s := string(*c)
	return &s
}

// ItemLotSource names which rule in the precedence chain produced an item's lot.
//
// The chain is ordered most specific first, and the source is reported alongside the lot so a planner can see why a SKU is being made in sixties rather than having to work it out from four places it might have come from.
type ItemLotSource string

const (
	// ItemLotSourceItemOverride indicates a lot size set on the item itself.
	ItemLotSourceItemOverride ItemLotSource = "item_override"
	// ItemLotSourceProductLine indicates the convention of the line the item sells under.
	ItemLotSourceProductLine ItemLotSource = "product_line"
	// ItemLotSourceDownstreamProductLine indicates a lot inherited from the finished goods an intermediate item becomes.
	ItemLotSourceDownstreamProductLine ItemLotSource = "downstream_product_line"
	// ItemLotSourceAccountDefault indicates the account-wide fallback lot size.
	ItemLotSourceAccountDefault ItemLotSource = "account_default"
	// ItemLotSourceNone indicates no rule supplied a lot.
	ItemLotSourceNone ItemLotSource = ""
)

func (s ItemLotSource) IsValid() bool {
	switch s {
	case ItemLotSourceItemOverride, ItemLotSourceProductLine,
		ItemLotSourceDownstreamProductLine, ItemLotSourceAccountDefault, ItemLotSourceNone:
		return true
	default:
		return false
	}
}

func (ItemLotSource) EnumValues() []string {
	return []string{
		string(ItemLotSourceItemOverride),
		string(ItemLotSourceProductLine),
		string(ItemLotSourceDownstreamProductLine),
		string(ItemLotSourceAccountDefault),
		string(ItemLotSourceNone),
	}
}

func (s *ItemLotSource) StringPtr() *string {
	if s == nil {
		return nil
	}
	value := string(*s)
	return &value
}

// MachineWorkStatus is what a machine is doing right now.
type MachineWorkStatus string

const (
	// MachineWorkStatusRunning indicates a released campaign with work still to scan.
	MachineWorkStatusRunning MachineWorkStatus = "running"
	// MachineWorkStatusIdle indicates nothing is released to the machine.
	MachineWorkStatusIdle MachineWorkStatus = "idle"
	// MachineWorkStatusDown indicates an open downtime event, which outranks running: a broken machine is not producing whatever the plan says.
	MachineWorkStatusDown MachineWorkStatus = "down"
)

func (s MachineWorkStatus) IsValid() bool {
	switch s {
	case MachineWorkStatusRunning, MachineWorkStatusIdle, MachineWorkStatusDown:
		return true
	default:
		return false
	}
}

func (MachineWorkStatus) EnumValues() []string {
	return []string{
		string(MachineWorkStatusRunning),
		string(MachineWorkStatusIdle),
		string(MachineWorkStatusDown),
	}
}

func (s *MachineWorkStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	value := string(*s)
	return &value
}
