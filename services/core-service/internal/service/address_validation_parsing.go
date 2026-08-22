package service

import (
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

type addressComponent struct {
	LongText  string   `json:"longText"`
	ShortText string   `json:"shortText"`
	Types     []string `json:"types"`
}

type postalAddress struct {
	RegionCode         string   `json:"regionCode"`
	AdministrativeArea string   `json:"administrativeArea"`
	Locality           string   `json:"locality"`
	Sublocality        string   `json:"sublocality"`
	PostalCode         string   `json:"postalCode"`
	AddressLines       []string `json:"addressLines"`
}

func normalizePlaceResourceName(placeID string) string {
	placeID = strings.TrimSpace(placeID)
	if strings.HasPrefix(placeID, "places/") {
		return placeID
	}
	return "places/" + placeID
}

func componentText(c addressComponent) string {
	if text := strings.TrimSpace(c.ShortText); text != "" {
		return text
	}
	return strings.TrimSpace(c.LongText)
}

func componentHasType(c addressComponent, want string) bool {
	for _, t := range c.Types {
		if t == want {
			return true
		}
	}
	return false
}

func findComponentText(components []addressComponent, types ...string) string {
	for _, want := range types {
		for _, c := range components {
			if componentHasType(c, want) {
				if text := componentText(c); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func parseAddressComponents(components []addressComponent) *domain.AddressComponents {
	result := &domain.AddressComponents{}

	streetNumber := findComponentText(components, "street_number")
	route := findComponentText(components, "route")
	premise := findComponentText(components, "premise")
	subpremise := findComponentText(components, "subpremise")

	switch {
	case streetNumber != "" && route != "":
		result.AddressLine1 = streetNumber + " " + route
	case route != "":
		result.AddressLine1 = route
	case streetNumber != "":
		result.AddressLine1 = streetNumber
	case premise != "":
		result.AddressLine1 = premise
	}

	if subpremise != "" {
		result.AddressLine2 = &subpremise
	}

	result.City = findComponentText(components,
		"locality",
		"postal_town",
		"sublocality_level_1",
		"sublocality",
		"administrative_area_level_2",
		"neighborhood",
	)
	result.State = findComponentText(components, "administrative_area_level_1")
	result.PostalCode = findComponentText(components, "postal_code")

	countryCode := findComponentText(components, "country")
	if countryCode != "" {
		result.CountryCode = countryCode
		result.Country = countryCode
	}

	return result
}

func applyPostalAddressFallback(result *domain.AddressComponents, postal *postalAddress) {
	if result == nil || postal == nil {
		return
	}

	if result.AddressLine1 == "" && len(postal.AddressLines) > 0 {
		result.AddressLine1 = strings.TrimSpace(postal.AddressLines[0])
		if len(postal.AddressLines) > 1 {
			line2 := strings.TrimSpace(strings.Join(postal.AddressLines[1:], ", "))
			if line2 != "" {
				result.AddressLine2 = &line2
			}
		}
	}

	if result.City == "" {
		result.City = firstNonEmptyString(
			strings.TrimSpace(postal.Locality),
			strings.TrimSpace(postal.Sublocality),
		)
	}
	if result.State == "" {
		result.State = strings.TrimSpace(postal.AdministrativeArea)
	}
	if result.PostalCode == "" {
		result.PostalCode = strings.TrimSpace(postal.PostalCode)
	}
	if result.CountryCode == "" && strings.TrimSpace(postal.RegionCode) != "" {
		result.CountryCode = strings.TrimSpace(postal.RegionCode)
	}
	if result.Country == "" && result.CountryCode != "" {
		result.Country = result.CountryCode
	}
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
