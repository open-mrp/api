package domain

// EstimateRateParams holds the parameters for estimating a shipping rate.
type EstimateRateParams struct {
	AccountID      string
	CarrierID      string
	ServiceLevelID string
	ProductLineIDs []string
	CustomerID     *string
	FromAddress    ShippingAddress
	ToAddress      ShippingAddress
	Parcels        []Parcel
	OrderTotal     *float64
	// Billing, when set, bills freight to a third party (matches Dashboard's
	// createShippingLine, which passes THIRD_PARTY billing to Shippo for
	// third-party-billed orders).
	Billing *ShippingBilling
}

// ShippingBilling carries third-party freight-billing details passed through to
// the carrier (Shippo) when the order is billed to a third party.
type ShippingBilling struct {
	// Type is the Shippo billing type, e.g. "THIRD_PARTY".
	Type    string
	Account string
	Country string
	Zip     string
}

// RateShopParams holds the parameters for rate shopping.
type RateShopParams struct {
	AccountID      string
	ProductLineIDs []string
	CustomerID     *string
	FromAddress    ShippingAddress
	ToAddress      ShippingAddress
	Parcels        []Parcel
	OrderTotal     *float64
}

// ShippingAddress is a simplified address for rate estimation.
type ShippingAddress struct {
	Name    string
	Company *string
	Street1 string
	Street2 *string
	City    string
	State   string
	Zip     string
	Country string
	Phone   *string
	Email   *string
	// Timezone is the stored IANA zone for this address, nil when it has not been resolved. Carried so a commitment can read a promised delivery instant as a local date rather than a UTC one.
	Timezone *string
}

// IsEmpty reports whether no meaningful address was provided, so callers can fall back to a resolved origin.
func (a ShippingAddress) IsEmpty() bool {
	return a.Street1 == "" && a.City == "" && a.State == "" && a.Zip == ""
}

// Parcel represents a package for rate estimation.
type Parcel struct {
	Weight string
	Length string
	Width  string
	Height string
}

// FetchShippingRateParams contains the parameters for fetching a shipping rate.
type FetchShippingRateParams struct {
	CarrierAccountObjectID string
	ServiceLevelToken      string
	FromAddress            ShippingAddress
	ToAddress              ShippingAddress
	Parcels                []Parcel
	// Billing, when set, bills freight to a third party (Shippo shipment extra).
	Billing *ShippingBilling
}

// FetchAllShippingRatesParams contains the parameters for fetching all shipping rates.
type FetchAllShippingRatesParams struct {
	CarrierAccountObjectID string
	FromAddress            ShippingAddress
	ToAddress              ShippingAddress
	Parcels                []Parcel
}

// RateShopResult holds the result of rate shopping.
type RateShopResult struct {
	Options       []*RateShopOption
	ExemptionType *string
	FlatRate      *float64
}

// RateShopOption represents a single carrier option with its rate.
type RateShopOption struct {
	CarrierID        string
	CarrierName      string
	ServiceLevelID   string
	ServiceLevelName string
	Rate             float64
	EstimatedDays    *int32
}

// ShippoRateOption represents a rate from a Shippo rate response.
type ShippoRateOption struct {
	ServiceLevelName  string
	ServiceLevelToken string
	Amount            float64
	EstimatedDays     *int32
}

// SSCC generation

// GenerateSSCC generates an SSCC-18 barcode string from a counter value.
func GenerateSSCC(counter int64) string {
	// Prefix "1" + 17-digit zero-padded counter = 18 digits total
	formatted := "1"
	s := formatInt64(counter, 17)
	return formatted + s
}

func formatInt64(n int64, width int) string {
	s := ""
	if n == 0 {
		s = "0"
	} else {
		for n > 0 {
			s = string(rune('0'+n%10)) + s
			n /= 10
		}
	}
	for len(s) < width {
		s = "0" + s
	}
	return s
}

// Carries everything needed to buy carrier labels for a shipment's cases.
type CreateLabelParams struct {
	CarrierAccountObjectID string
	ServiceLevelToken      string
	FromAddress            ShippingAddress
	ToAddress              ShippingAddress
	// Holds one entry per shipping case, in case order; the result packages match this order.
	Parcels []Parcel
	Billing *ShippingBilling
}

// Reports the outcome of a label purchase across a shipment's cases.
type LabelResult struct {
	MasterTrackingNumber string
	NegotiatedRate       float64
	// Holds one purchased label per parcel, in the same order as CreateLabelParams.Parcels.
	Packages []LabelPackage
}

// Holds a single case's purchased label.
type LabelPackage struct {
	TrackingNumber      string
	LabelURL            string
	ShippoTransactionID string
}
