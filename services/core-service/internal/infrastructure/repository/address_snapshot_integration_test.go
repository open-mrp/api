//go:build integration

package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
)

// TestAddressSnapshotBackfill_CopiesTheSourceAddress runs the backfill over a scratch document table with the
// snapshot column convention, in small batches.
func TestAddressSnapshotBackfill_CopiesTheSourceAddress(t *testing.T) {
	pool := testDB(t)
	table := fmt.Sprintf("snapshot_backfill_test_%d", time.Now().UnixNano())
	lcExec(t, pool, "CREATE TABLE `"+table+"` (id varchar(191) NOT NULL PRIMARY KEY, billing_address_id varchar(191) NULL, "+
		"bill_to_address_id varchar(191) NULL, bill_to_name varchar(255) NULL, bill_to_line_1 varchar(255) NULL, bill_to_line_2 varchar(255) NULL, "+
		"bill_to_city varchar(255) NULL, bill_to_state varchar(255) NULL, bill_to_postal_code varchar(255) NULL, bill_to_country char(2) NULL, "+
		"bill_to_phone varchar(255) NULL, bill_to_email varchar(255) NULL) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")
	t.Cleanup(func() { _, _ = pool.Exec("DROP TABLE `" + table + "`") })

	var addrs []string
	rows, err := pool.Query("SELECT id FROM address ORDER BY id LIMIT 2")
	require.NoError(t, err)
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		addrs = append(addrs, id)
	}
	require.NoError(t, rows.Close())
	require.Len(t, addrs, 2, "the seed has addresses")

	lcExec(t, pool, "INSERT INTO `"+table+"` (id, billing_address_id) VALUES ('doc_1', ?), ('doc_2', ?), ('doc_3', NULL), ('doc_4', ?)",
		addrs[0], addrs[1], addrs[0])
	lcExec(t, pool, "UPDATE `"+table+"` SET bill_to_name = 'Kept', bill_to_country = 'ZZ' WHERE id = 'doc_4'")

	q := sqlc.New(pool)
	b, err := NewAddressSnapshotBackfill(q, q, table, "billing_address_id", "bill_to")
	require.NoError(t, err)
	ctx := context.Background()
	cursor, batches := "", 0
	for {
		next, _, done, err := b.Batch(ctx, &backfill.Meter{}, cursor, 2)
		require.NoError(t, err)
		cursor, batches = next, batches+1
		if done {
			break
		}
		require.Less(t, batches, 10)
	}
	assert.Equal(t, "doc_4", cursor)

	matches := func(docID string) bool {
		var ok bool
		require.NoError(t, pool.QueryRow(`SELECT d.bill_to_address_id <=> a.id AND d.bill_to_name <=> a.name AND d.bill_to_line_1 <=> g.street_line_1
			AND d.bill_to_line_2 <=> g.street_line_2 AND d.bill_to_city <=> g.locality AND d.bill_to_state <=> g.state
			AND d.bill_to_postal_code <=> g.postal_code AND d.bill_to_country <=> g.country AND d.bill_to_phone <=> a.phone
			AND d.bill_to_email <=> a.email
			FROM `+"`"+table+"`"+` d JOIN address a ON a.id = d.billing_address_id JOIN geolocation g ON g.id = a.geolocation_id WHERE d.id = ?`,
			docID).Scan(&ok))
		return ok
	}
	assert.True(t, matches("doc_1"))
	assert.True(t, matches("doc_2"))

	var country gosql.NullString
	require.NoError(t, pool.QueryRow("SELECT bill_to_country FROM `"+table+"` WHERE id = 'doc_3'").Scan(&country))
	assert.False(t, country.Valid, "a document with no address gets no snapshot")

	var name string
	require.NoError(t, pool.QueryRow("SELECT bill_to_name FROM `"+table+"` WHERE id = 'doc_4'").Scan(&name))
	assert.Equal(t, "Kept", name, "an existing snapshot is never overwritten")
}
