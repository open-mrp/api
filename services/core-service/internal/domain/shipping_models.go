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
