package id

import (
	apierror "github.com/open-mrp/api/shared/errors"

	nanoid "github.com/matoous/go-nanoid/v2"
)

// IDLength is the length of the ID. The longer the ID, the less likely there will be a collision.
//   - IDLength12 - 308M IDs
//   - IDLength19 - 86T IDs
//   - IDLength22 - 18,660T IDs
type IDLength int

const (
	IDLength12 IDLength = 12
	IDLength19 IDLength = 19
	IDLength22 IDLength = 22
)

const (
	charset = "0123456789abcdefghijklmnopqrstuvwxyz"
)

// MaxLength is the longest ID GenID returns for any prefix. Columns sized to it rather than
// varchar(191) (sales_line_fact's, to fit a wide covering key) would reject a longer one.
const MaxLength = 64

// genNanoID generates a new nano ID with the given length.
func genNanoID(length IDLength) (string, *apierror.APIError) {
	id, err := nanoid.Generate(charset, int(length))
	if err != nil {
		return "", apierror.NewInternalError(err, "Failed to generate nano ID.")
	}
	return id, nil
}
