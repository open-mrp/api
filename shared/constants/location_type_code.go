package constants

// LocationTypeCode represents the type of a location.
type LocationTypeCode string

const (
	// LocationTypeCodeBuilding indicates a building-level location.
	LocationTypeCodeBuilding LocationTypeCode = "building"
	// LocationTypeCodeSection indicates a section within a building.
	LocationTypeCodeSection LocationTypeCode = "section"
	// LocationTypeCodeAisle indicates an aisle within a section.
	LocationTypeCodeAisle LocationTypeCode = "aisle"
	// LocationTypeCodeRack indicates a rack within an aisle.
	LocationTypeCodeRack LocationTypeCode = "rack"
	// LocationTypeCodeShelf indicates a shelf within a rack.
	LocationTypeCodeShelf LocationTypeCode = "shelf"
	// LocationTypeCodeBin indicates a bin within a shelf.
	LocationTypeCodeBin LocationTypeCode = "bin"
)

func (s LocationTypeCode) IsValid() bool {
	switch s {
	case LocationTypeCodeBuilding, LocationTypeCodeSection, LocationTypeCodeAisle, LocationTypeCodeRack, LocationTypeCodeShelf, LocationTypeCodeBin:
		return true
	default:
		return false
	}
}

func (s LocationTypeCode) EnumValues() []string {
	return []string{string(LocationTypeCodeBuilding), string(LocationTypeCodeSection), string(LocationTypeCodeAisle), string(LocationTypeCodeRack), string(LocationTypeCodeShelf), string(LocationTypeCodeBin)}
}
