package constants

// ActivationStatus is whether a configuration row is currently applied.
//
// Modelled as a status rather than an `is_active` boolean so that a third state — say `scheduled` or `expired` — can be added without changing the field's shape or meaning.
type ActivationStatus string

const (
	// ActivationStatusActive indicates the row is applied.
	ActivationStatusActive ActivationStatus = "active"
	// ActivationStatusInactive indicates the row exists but is not applied.
	ActivationStatusInactive ActivationStatus = "inactive"
)

func (s ActivationStatus) IsValid() bool {
	switch s {
	case ActivationStatusActive, ActivationStatusInactive:
		return true
	default:
		return false
	}
}

func (s ActivationStatus) EnumValues() []string {
	return []string{string(ActivationStatusActive), string(ActivationStatusInactive)}
}

// ActivationStatusOf maps the stored flag onto the enum the API exposes.
func ActivationStatusOf(active bool) ActivationStatus {
	if active {
		return ActivationStatusActive
	}
	return ActivationStatusInactive
}

// DemandOverrideScope is what an override targets.
type DemandOverrideScope string

const (
	// DemandOverrideScopeItem indicates the override targets a single item.
	DemandOverrideScopeItem DemandOverrideScope = "item"
	// DemandOverrideScopeProductLine indicates the override targets a product line, distributed across its items.
	DemandOverrideScopeProductLine DemandOverrideScope = "product_line"
)

func (s DemandOverrideScope) IsValid() bool {
	switch s {
	case DemandOverrideScopeItem, DemandOverrideScopeProductLine:
		return true
	default:
		return false
	}
}

func (s DemandOverrideScope) EnumValues() []string {
	return []string{string(DemandOverrideScopeItem), string(DemandOverrideScopeProductLine)}
}

// DemandOverrideAdjustment is how an override's value changes the forecast. When several land on the same month they apply in declaration order.
type DemandOverrideAdjustment string

const (
	// DemandOverrideAdjustmentAbsolute replaces the forecast for the period.
	DemandOverrideAdjustmentAbsolute DemandOverrideAdjustment = "absolute"
	// DemandOverrideAdjustmentDeltaUnits adds to the forecast.
	DemandOverrideAdjustmentDeltaUnits DemandOverrideAdjustment = "delta_units"
	// DemandOverrideAdjustmentDeltaPercent scales the forecast.
	DemandOverrideAdjustmentDeltaPercent DemandOverrideAdjustment = "delta_percent"
)

func (a DemandOverrideAdjustment) IsValid() bool {
	switch a {
	case DemandOverrideAdjustmentAbsolute, DemandOverrideAdjustmentDeltaUnits, DemandOverrideAdjustmentDeltaPercent:
		return true
	default:
		return false
	}
}

func (a DemandOverrideAdjustment) EnumValues() []string {
	return []string{
		string(DemandOverrideAdjustmentAbsolute),
		string(DemandOverrideAdjustmentDeltaUnits),
		string(DemandOverrideAdjustmentDeltaPercent),
	}
}

func (s *ActivationStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (s *DemandOverrideScope) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (a *DemandOverrideAdjustment) StringPtr() *string {
	if a == nil {
		return nil
	}
	str := string(*a)
	return &str
}
