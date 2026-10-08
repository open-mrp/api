package repository

import (
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/blobstore"
)

// Payloads is where repositories keep documents too large for their rows. A nil *Payloads keeps
// every document inline.
type Payloads struct {
	// Store (required) holds the documents.
	Store *blobstore.Store
	// Pool (required) runs the writes that follow a transaction's commit, when the transaction's own
	// queries are already closed.
	Pool *sqlc.Queries
}

func (p *Payloads) store() *blobstore.Store {
	if p == nil {
		return nil
	}
	return p.Store
}
