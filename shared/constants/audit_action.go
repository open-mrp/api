package constants

// AuditAction is the type of mutation that occurred for an audited resource.
type AuditAction string

const (
	// AuditActionCreate represents a creation mutation.
	AuditActionCreate AuditAction = "create"
	// AuditActionUpdate represents an update mutation.
	AuditActionUpdate AuditAction = "update"
	// AuditActionUpsert represents an upsert mutation.
	AuditActionUpsert AuditAction = "upsert"
	// AuditActionDelete represents a deletion mutation.
	AuditActionDelete AuditAction = "delete"
	// AuditActionRestore represents a restore mutation.
	AuditActionRestore AuditAction = "restore"
	// AuditActionArchive represents an archive mutation.
	AuditActionArchive AuditAction = "archive"
	// AuditActionApprove represents a human approving a gated action (e.g. letting a review-gated agent tool run).
	AuditActionApprove AuditAction = "approve"
	// AuditActionDeny represents a human denying a gated action (e.g. rejecting a review-gated agent tool).
	AuditActionDeny AuditAction = "deny"
)

func (m AuditAction) IsValid() bool {
	switch m {
	case AuditActionCreate, AuditActionUpdate, AuditActionUpsert, AuditActionDelete,
		AuditActionRestore, AuditActionArchive, AuditActionApprove, AuditActionDeny:
		return true
	default:
		return false
	}
}

func (m AuditAction) EnumValues() []string {
	return []string{
		string(AuditActionCreate),
		string(AuditActionUpdate),
		string(AuditActionUpsert),
		string(AuditActionDelete),
		string(AuditActionRestore),
		string(AuditActionArchive),
		string(AuditActionApprove),
		string(AuditActionDeny),
	}
}
