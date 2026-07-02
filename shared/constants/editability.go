package constants

// Editability is whether the caller can edit a resource. It is an enum (not a boolean) so additional states (e.g. locked, restricted) can be added without a breaking change.
type Editability string

const (
	// EditabilityEditable means the caller can edit the resource.
	EditabilityEditable Editability = "editable"
	// EditabilityReadOnly means the caller cannot edit the resource.
	EditabilityReadOnly Editability = "read_only"
)

func (s Editability) IsValid() bool {
	switch s {
	case EditabilityEditable, EditabilityReadOnly:
		return true
	default:
		return false
	}
}

func (s Editability) EnumValues() []string {
	return []string{string(EditabilityEditable), string(EditabilityReadOnly)}
}

func (s *Editability) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// EditabilityFromBool maps the persisted boolean to its editability enum.
func EditabilityFromBool(editable bool) Editability {
	if editable {
		return EditabilityEditable
	}
	return EditabilityReadOnly
}
