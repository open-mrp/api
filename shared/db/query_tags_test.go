package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"sync"
	"testing"

	"github.com/XSAM/otelsql"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/apikit/querytag"
)

// recordingDriver records the SQL that reaches it. skipDirect makes it refuse direct execution, as the MySQL driver does when it cannot interpolate, so database/sql falls back to prepare.
type recordingDriver struct {
	mu         sync.Mutex
	seen       []string
	skipDirect bool
}

func (d *recordingDriver) record(q string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen = append(d.seen, q)
}

func (d *recordingDriver) Open(string) (driver.Conn, error) { return &recordingConn{d: d}, nil }
func (d *recordingDriver) Connect(context.Context) (driver.Conn, error) {
	return &recordingConn{d: d}, nil
}
func (d *recordingDriver) Driver() driver.Driver { return d }

type recordingConn struct{ d *recordingDriver }

func (c *recordingConn) Prepare(q string) (driver.Stmt, error) {
	c.d.record(q)
	return recordingStmt{}, nil
}
func (c *recordingConn) Close() error              { return nil }
func (c *recordingConn) Begin() (driver.Tx, error) { return recordingTx{}, nil }
func (c *recordingConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return recordingTx{}, nil
}

func (c *recordingConn) ExecContext(_ context.Context, q string, _ []driver.NamedValue) (driver.Result, error) {
	if c.d.skipDirect {
		return nil, driver.ErrSkip
	}
	c.d.record(q)
	return driver.RowsAffected(0), nil
}

func (c *recordingConn) QueryContext(_ context.Context, q string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.d.skipDirect {
		return nil, driver.ErrSkip
	}
	c.d.record(q)
	return emptyRows{}, nil
}

type recordingStmt struct{}

func (recordingStmt) Close() error                               { return nil }
func (recordingStmt) NumInput() int                              { return -1 }
func (recordingStmt) Exec([]driver.Value) (driver.Result, error) { return driver.RowsAffected(0), nil }
func (recordingStmt) Query([]driver.Value) (driver.Rows, error)  { return emptyRows{}, nil }

type recordingTx struct{}

func (recordingTx) Commit() error   { return nil }
func (recordingTx) Rollback() error { return nil }

type emptyRows struct{}

func (emptyRows) Columns() []string         { return []string{"x"} }
func (emptyRows) Close() error              { return nil }
func (emptyRows) Next([]driver.Value) error { return io.EOF }

func TestTaggingConnectorTagsEveryPath(t *testing.T) {
	const want = "SELECT 1 /*app='core-service',job='sweep'*/"
	for name, open := range map[string]func(driver.Connector) *sql.DB{
		"plain":   sql.OpenDB,
		"otelsql": func(c driver.Connector) *sql.DB { return otelsql.OpenDB(c) },
	} {
		for _, skip := range []bool{false, true} {
			t.Run(name, func(t *testing.T) {
				rec := &recordingDriver{skipDirect: skip}
				pool := open(taggingConnector{base: rec, static: map[string]string{querytag.App: "core-service"}})
				defer pool.Close()
				ctx := querytag.With(context.Background(), querytag.Job, "sweep")

				rows, err := pool.QueryContext(ctx, "SELECT 1")
				require.NoError(t, err)
				require.NoError(t, rows.Close())
				_, err = pool.ExecContext(ctx, "SELECT 1;")
				require.NoError(t, err)
				tx, err := pool.BeginTx(ctx, nil)
				require.NoError(t, err)
				_, err = tx.ExecContext(ctx, "SELECT 1")
				require.NoError(t, err)
				require.NoError(t, tx.Commit())

				require.Equal(t, []string{want, want + ";", want}, rec.seen)
			})
		}
	}
}

func TestTaggingConnectorWithoutTagsLeavesSQLAlone(t *testing.T) {
	rec := &recordingDriver{}
	pool := sql.OpenDB(taggingConnector{base: rec})
	defer pool.Close()
	_, err := pool.ExecContext(context.Background(), "SELECT 1")
	require.NoError(t, err)
	require.Equal(t, []string{"SELECT 1"}, rec.seen)
}
