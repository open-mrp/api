package constants

// AuditAction is the type of mutation that occurred for an audited resource.
type AuditAction string

const (
	// AuditActionCreate represents a creation mutation.
	AuditActionCreate AuditAction = "create"
	// AuditActionUpdate represents an update mutation.
	AuditActionUpdate AuditAction = "update"
	// AuditActionDelete represents a deletion mutation.
	AuditActionDelete AuditAction = "delete"
	// AuditActionRestore represents a restore mutation.
	AuditActionRestore AuditAction = "restore"
	// AuditActionArchive represents an archive mutation.
	AuditActionArchive AuditAction = "archive"
)

func (m AuditAction) IsValid() bool {
	switch m {
	case AuditActionCreate, AuditActionUpdate, AuditActionDelete,
		AuditActionRestore, AuditActionArchive:
		return true
	default:
		return false
	}
}

func (m AuditAction) EnumValues() []string {
	return []string{
		string(AuditActionCreate),
		string(AuditActionUpdate),
		string(AuditActionDelete),
		string(AuditActionRestore),
		string(AuditActionArchive),
	}
}
