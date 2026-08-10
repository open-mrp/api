package domain

import "github.com/augno/api/shared/messaging"

// RepoFactory builds repositories for the logging service.
type RepoFactory interface {
	NewRequestLogRepo() RequestLogRepo
	NewAuditEventRepo() AuditEventRepo
	NewOutboxRepo() messaging.OutboxRepo
}
