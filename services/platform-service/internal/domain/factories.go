package domain

// RepoFactory builds repositories for the logging service.
type RepoFactory interface {
	NewRequestLogRepo() RequestLogRepo
	NewAuditEventRepo() AuditEventRepo
}
