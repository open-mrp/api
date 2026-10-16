package constants

// ManufacturingMethodStatus is where a manufacturing method version is in its life. Only drafts are editable.
type ManufacturingMethodStatus string

const (
	// ManufacturingMethodStatusDraft is editable and not used for production.
	ManufacturingMethodStatusDraft ManufacturingMethodStatus = "draft"
	// ManufacturingMethodStatusActive is the version used for new production of its item.
	ManufacturingMethodStatusActive ManufacturingMethodStatus = "active"
	// ManufacturingMethodStatusArchived is a previous version, kept for history.
	ManufacturingMethodStatusArchived ManufacturingMethodStatus = "archived"
)

func (s ManufacturingMethodStatus) IsValid() bool {
	switch s {
	case ManufacturingMethodStatusDraft, ManufacturingMethodStatusActive, ManufacturingMethodStatusArchived:
		return true
	default:
		return false
	}
}

func (s ManufacturingMethodStatus) EnumValues() []string {
	return []string{string(ManufacturingMethodStatusDraft), string(ManufacturingMethodStatusActive), string(ManufacturingMethodStatusArchived)}
}

func (s *ManufacturingMethodStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// OperationSequence says when an operation starts relative to the one before it.
type OperationSequence string

const (
	// OperationSequenceAfterPrevious starts once the previous operation is done.
	OperationSequenceAfterPrevious OperationSequence = "after_previous"
	// OperationSequenceWithPrevious runs alongside the previous operation.
	OperationSequenceWithPrevious OperationSequence = "with_previous"
)

func (s OperationSequence) IsValid() bool {
	switch s {
	case OperationSequenceAfterPrevious, OperationSequenceWithPrevious:
		return true
	default:
		return false
	}
}

func (s OperationSequence) EnumValues() []string {
	return []string{string(OperationSequenceAfterPrevious), string(OperationSequenceWithPrevious)}
}

func (s *OperationSequence) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// OperationType is who does an operation's work. Only inside work exists today; outside (subcontracted) work is
// planned.
type OperationType string

const (
	// OperationTypeInside is done in-house at a work center.
	OperationTypeInside OperationType = "inside"
)

func (t OperationType) IsValid() bool {
	return t == OperationTypeInside
}

func (t OperationType) EnumValues() []string {
	return []string{string(OperationTypeInside)}
}

func (t *OperationType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}
