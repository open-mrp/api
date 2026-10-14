package domain

// AddressSnapshot is an address as a document printed it (an order's bill_to or ship_to), copied when the document
// is created so editing the saved address never rewrites the document.
type AddressSnapshot struct {
	// AddressID is the saved address it was copied from; nil for an address given inline.
	AddressID  *string
	Name       string
	Line1      *string
	Line2      *string
	City       *string
	State      *string
	PostalCode *string
	// Country is the ISO 3166-1 alpha-2 code.
	Country string
	Phone   *string
	Email   *string
}

// SnapshotAddress copies a saved address for a document; nil stays nil.
func SnapshotAddress(a *Address) *AddressSnapshot {
	if a == nil {
		return nil
	}
	id := a.ID
	s := &AddressSnapshot{AddressID: &id, Name: a.Name, Phone: copyString(a.Phone), Email: copyString(a.Email)}
	if g := a.Geolocation; g != nil {
		s.Line1 = copyString(g.StreetLine1)
		s.Line2 = copyString(g.StreetLine2)
		s.City = copyString(g.Locality)
		s.State = copyString(g.State)
		s.PostalCode = copyString(g.PostalCode)
		s.Country = g.Country
	}
	return s
}

func copyString(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}
