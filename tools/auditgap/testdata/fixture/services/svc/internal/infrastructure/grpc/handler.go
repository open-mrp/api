package grpc

import (
	"example.com/fixture/services/svc/internal/infrastructure/sqlc"
	"example.com/fixture/shared/audit"
)

type UnimplementedThingServer struct{}

func (UnimplementedThingServer) Audited()    {}
func (UnimplementedThingServer) Unaudited()  {}
func (UnimplementedThingServer) Indirect()   {}
func (UnimplementedThingServer) ReadOnly()   {}
func (UnimplementedThingServer) OutboxOnly() {}
func (UnimplementedThingServer) Exempt()     {}

type ThingServer interface {
	Audited()
	Unaudited()
	Indirect()
	ReadOnly()
	OutboxOnly()
	Exempt()
}

func Register(register func(ThingServer), q *sqlc.Queries) { register(&handler{q: q}) }

type handler struct {
	UnimplementedThingServer
	q *sqlc.Queries
}

func (h *handler) Audited() {
	h.q.CreateThing()
	audit.Publisher{}.Publish()
}

func (h *handler) Unaudited() { h.q.CreateThing() }

func (h *handler) Indirect() { h.write(func() { h.q.CreateThing() }) }

func (h *handler) write(fn func()) { fn() }

func (h *handler) ReadOnly() { h.q.GetThing() }

func (h *handler) OutboxOnly() { h.q.CreateOutboxMessage() }

func (h *handler) Exempt() { h.q.CreateThing() }

// NotServed writes but is not a gRPC method, so it is not a root.
func (h *handler) NotServed() { h.q.CreateThing() }
