package db

import (
	"context"
	"database/sql/driver"
	"time"

	"github.com/open-mrp/apikit/querytag"
)

// taggingConnector appends each statement's SQLCommenter tags (querytag) as it reaches the driver, beneath otelsql, so every path is tagged: queries, execs, prepares and everything inside a transaction.
type taggingConnector struct {
	base   driver.Connector
	static map[string]string
	// maxQueryTime bounds each SELECT on the database side (see withQueryTimeout); zero leaves them unbounded.
	maxQueryTime time.Duration
}

func (c taggingConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &taggingConn{Conn: conn, static: c.static, maxQueryTime: c.maxQueryTime}, nil
}

func (c taggingConnector) Driver() driver.Driver { return c.base.Driver() }

// taggingConn forwards every optional interface the MySQL driver's connection implements. database/sql detects them by type assertion, so a missing one would silently change behaviour (e.g. prepare every query instead of interpolating).
type taggingConn struct {
	driver.Conn
	static       map[string]string
	maxQueryTime time.Duration
}

func (c *taggingConn) tag(ctx context.Context, query string) string {
	return querytag.Append(withQueryTimeout(ctx, query, c.maxQueryTime), querytag.Comment(ctx, c.static))
}

func (c *taggingConn) Prepare(query string) (driver.Stmt, error) {
	return c.Conn.Prepare(c.tag(context.Background(), query))
}

func (c *taggingConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if p, ok := c.Conn.(driver.ConnPrepareContext); ok {
		return p.PrepareContext(ctx, c.tag(ctx, query))
	}
	return c.Conn.Prepare(c.tag(ctx, query))
}

func (c *taggingConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if e, ok := c.Conn.(driver.ExecerContext); ok {
		return e.ExecContext(ctx, c.tag(ctx, query), args)
	}
	return nil, driver.ErrSkip
}

func (c *taggingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if q, ok := c.Conn.(driver.QueryerContext); ok {
		return q.QueryContext(ctx, c.tag(ctx, query), args)
	}
	return nil, driver.ErrSkip
}

func (c *taggingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if b, ok := c.Conn.(driver.ConnBeginTx); ok {
		return b.BeginTx(ctx, opts)
	}
	//lint:ignore SA1019 fallback for drivers without BeginTx, as database/sql does
	return c.Conn.Begin() //nolint:staticcheck // same, for golangci-lint
}

func (c *taggingConn) Ping(ctx context.Context) error {
	if p, ok := c.Conn.(driver.Pinger); ok {
		return p.Ping(ctx)
	}
	return nil
}

func (c *taggingConn) ResetSession(ctx context.Context) error {
	if r, ok := c.Conn.(driver.SessionResetter); ok {
		return r.ResetSession(ctx)
	}
	return nil
}

func (c *taggingConn) IsValid() bool {
	if v, ok := c.Conn.(driver.Validator); ok {
		return v.IsValid()
	}
	return true
}

func (c *taggingConn) CheckNamedValue(nv *driver.NamedValue) error {
	if n, ok := c.Conn.(driver.NamedValueChecker); ok {
		return n.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}
