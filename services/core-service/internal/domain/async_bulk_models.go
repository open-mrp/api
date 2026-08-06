package domain

// BulkOperationJobEvent is the message every async bulk operation enqueues: only the
// job's ID. The resolved payload lives on the job row, and the account comes from the
// identity restored alongside the message — so there is exactly one copy of each.
//
// The AMQP body is marshaled and unmarshaled by our own producer and consumer against
// this same type, so it carries no wire tags: serialization lives at the boundary, the
// domain stays a plain value object.
type BulkOperationJobEvent struct {
	JobID string
}
