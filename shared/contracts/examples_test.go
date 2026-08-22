package contracts_test

import (
	"github.com/open-mrp/api/shared/contracts"
)

// ExampleNewGRPCClientConn shows the minimal configuration for dialing another
// service: only the target is required; a nil config receives production
// defaults.
func ExampleNewGRPCClientConn() {
	conn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{
		URL:  "auth-service:9092",
		Name: "auth-service",
	}, nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
}
