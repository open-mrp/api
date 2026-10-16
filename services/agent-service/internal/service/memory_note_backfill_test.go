package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/client"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
)

func memoryRow(id, category, entityType, entityID string, importance float64) sqlc.ListAgentMemoriesAfterRow {
	at := pgtype.Timestamptz{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	r := sqlc.ListAgentMemoriesAfterRow{ID: id, AccountID: "ac_1", Category: category, Content: "c " + id, Importance: importance, CreatedAt: at, UpdatedAt: at}
	if entityType != "" {
		r.EntityType = pgtype.Text{String: entityType, Valid: true}
		r.EntityID = pgtype.Text{String: entityID, Valid: true}
	}
	return r
}

func TestMemoryAsNote(t *testing.T) {
	t.Parallel()
	n := memoryAsNote(memoryRow("agmm_1", "preference", "customer", "ac_c", 0.8))
	assert.Equal(t, "preference", n.Type)
	assert.Equal(t, "high", n.Importance)
	assert.Equal(t, "customer", n.RecordType)
	assert.Equal(t, "ac_c", n.RecordID)
	assert.True(t, strings.HasPrefix(n.ID, "nt_"))
	assert.Equal(t, n.ID, memoryAsNote(memoryRow("agmm_1", "fact", "", "", 0)).ID, "the same memory always becomes the same note")

	product := memoryAsNote(memoryRow("agmm_2", "fact", "product", "it_1", 0.5))
	assert.Equal(t, "item", product.RecordType, "products are items")
	assert.Equal(t, "normal", product.Importance)

	account := memoryAsNote(memoryRow("agmm_3", "instruction", "account", "ac_1", 0))
	assert.Empty(t, account.RecordID, "a memory on the account is account-wide")
	assert.Equal(t, "low", account.Importance)

	unknown := memoryAsNote(memoryRow("agmm_4", "something", "widget", "w_1", 0.4))
	assert.Empty(t, unknown.RecordID, "an entity no record type names is account-wide")
	assert.Equal(t, "fact", unknown.Type)
}

func TestMemoryNoteBackfill_PagesAndFinishes(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	core := clientmock.NewMockCoreClient(ctrl)
	rows := []sqlc.ListAgentMemoriesAfterRow{memoryRow("agmm_1", "fact", "", "", 0), memoryRow("agmm_2", "fact", "", "", 0)}
	b := NewMemoryNoteBackfill(func(_ context.Context, after string, limit int) ([]sqlc.ListAgentMemoriesAfterRow, error) {
		var out []sqlc.ListAgentMemoriesAfterRow
		for _, r := range rows {
			if r.ID > after && len(out) < limit {
				out = append(out, r)
			}
		}
		return out, nil
	}, core)

	core.EXPECT().ImportNotes(gomock.Any(), gomock.Len(2)).Return(int64(2), nil)
	next, n, done, err := b.Batch(context.Background(), &backfill.Meter{}, "", 2)
	require.NoError(t, err)
	assert.Equal(t, "agmm_2", next)
	assert.Equal(t, 2, n)
	assert.False(t, done, "a full page may have more behind it")

	next, n, done, err = b.Batch(context.Background(), &backfill.Meter{}, next, 2)
	require.NoError(t, err)
	assert.Equal(t, "agmm_2", next)
	assert.Zero(t, n)
	assert.True(t, done)
}

func TestAppendRecalledNotes(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "base", appendRecalledNotes("base", nil))
	got := appendRecalledNotes("base", []domain.RecalledNote{
		{Type: "instruction", Body: "Confirm freight."},
		{Type: "fact", Body: "Orders monthly.", RecordType: "customer", RecordID: "ac_c"},
	})
	assert.Contains(t, got, "- [instruction] Confirm freight.\n")
	assert.Contains(t, got, "- [fact, on customer ac_c] Orders monthly.\n")
}
