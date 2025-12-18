package id

import (
	"github.com/augno/api/shared/contracts"

	nanoid "github.com/matoous/go-nanoid/v2"
)

// IDLength is the length of the ID. The longer the ID, the less likely there will
// be a collision.
// - IDLength12: 308M IDs
// - IDLength19: 86T IDs
// - IDLength22: 18,660T IDs
type IDLength int

const (
	IDLength12 IDLength = 12
	IDLength19 IDLength = 19
	IDLength22 IDLength = 22
)

const (
	charset = "0123456789abcdefghijklmnopqrstuvwxyz"
)

func genNanoID(length IDLength) (string, *contracts.APIError) {
	id, err := nanoid.Generate(charset, int(length))
	if err != nil {
		return "", contracts.NewInternalError(err, "Failed to generate nano ID.")
	}
	return id, nil
}
