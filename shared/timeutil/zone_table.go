package timeutil

// subdivisionZones maps a subdivision onto its zone, for the countries that span more than one.
//
// A state split across two zones is filed under the zone most of its population keeps, because the alternative — refusing to answer — leaves the address on the account default, which is wrong more often. The thirteen split US states are AZ, FL, ID, IN, KS, KY, MI, ND, NE, OR, SD, TN and TX; an address in the minority half is corrected by storing its zone on the geolocation.
var subdivisionZones = map[string]map[string]string{
	"US": {
		// Eastern
		"CT": "America/New_York", "DC": "America/New_York", "DE": "America/New_York",
		"FL": "America/New_York", "GA": "America/New_York", "IN": "America/New_York",
		"KY": "America/New_York", "MA": "America/New_York", "MD": "America/New_York",
		"ME": "America/New_York", "MI": "America/New_York", "NC": "America/New_York",
		"NH": "America/New_York", "NJ": "America/New_York", "NY": "America/New_York",
		"OH": "America/New_York", "PA": "America/New_York", "RI": "America/New_York",
		"SC": "America/New_York", "VA": "America/New_York", "VT": "America/New_York",
		"WV": "America/New_York",
		// Central
		"AL": "America/Chicago", "AR": "America/Chicago", "IA": "America/Chicago",
		"IL": "America/Chicago", "KS": "America/Chicago", "LA": "America/Chicago",
		"MN": "America/Chicago", "MO": "America/Chicago", "MS": "America/Chicago",
		"ND": "America/Chicago", "NE": "America/Chicago", "OK": "America/Chicago",
		"SD": "America/Chicago", "TN": "America/Chicago", "TX": "America/Chicago",
		"WI": "America/Chicago",
		// Mountain. Arizona keeps no daylight saving, which is why it cannot share Denver's zone.
		"AZ": "America/Phoenix", "CO": "America/Denver", "ID": "America/Denver",
		"MT": "America/Denver", "NM": "America/Denver", "UT": "America/Denver",
		"WY": "America/Denver",
		// Pacific and beyond
		"CA": "America/Los_Angeles", "NV": "America/Los_Angeles", "OR": "America/Los_Angeles",
		"WA": "America/Los_Angeles", "AK": "America/Anchorage", "HI": "Pacific/Honolulu",
		// Territories
		"AS": "Pacific/Pago_Pago", "GU": "Pacific/Guam", "MP": "Pacific/Saipan",
		"PR": "America/Puerto_Rico", "VI": "America/St_Thomas",
	},
	"CA": {
		"AB": "America/Edmonton", "BC": "America/Vancouver", "MB": "America/Winnipeg",
		"NB": "America/Halifax", "NL": "America/St_Johns", "NS": "America/Halifax",
		"NT": "America/Yellowknife", "NU": "America/Iqaluit", "ON": "America/Toronto",
		"PE": "America/Halifax", "QC": "America/Toronto", "YT": "America/Whitehorse",
		// Saskatchewan keeps no daylight saving.
		"SK": "America/Regina",
	},
	"MX": {
		"BC": "America/Tijuana", "BS": "America/Mazatlan", "CHH": "America/Chihuahua",
		"SIN": "America/Mazatlan", "SON": "America/Hermosillo", "NAY": "America/Mazatlan",
		"ROO": "America/Cancun",
	},
	"AU": {
		"ACT": "Australia/Sydney", "NSW": "Australia/Sydney", "NT": "Australia/Darwin",
		"QLD": "Australia/Brisbane", "SA": "Australia/Adelaide", "TAS": "Australia/Hobart",
		"VIC": "Australia/Melbourne", "WA": "Australia/Perth",
	},
}

// subdivisionNames aliases spelled-out subdivisions onto their codes, because the state column is free text and the same place arrives spelled out as often as abbreviated.
var subdivisionNames = map[string]map[string]string{
	"US": {
		"ALABAMA": "AL", "ALASKA": "AK", "ARIZONA": "AZ", "ARKANSAS": "AR",
		"CALIFORNIA": "CA", "COLORADO": "CO", "CONNECTICUT": "CT", "DELAWARE": "DE",
		"DISTRICT OF COLUMBIA": "DC", "WASHINGTON DC": "DC",
		"FLORIDA": "FL", "GEORGIA": "GA", "HAWAII": "HI", "IDAHO": "ID",
		"ILLINOIS": "IL", "INDIANA": "IN", "IOWA": "IA", "KANSAS": "KS",
		"KENTUCKY": "KY", "LOUISIANA": "LA", "MAINE": "ME", "MARYLAND": "MD",
		"MASSACHUSETTS": "MA", "MICHIGAN": "MI", "MINNESOTA": "MN", "MISSISSIPPI": "MS",
		"MISSOURI": "MO", "MONTANA": "MT", "NEBRASKA": "NE", "NEVADA": "NV",
		"NEW HAMPSHIRE": "NH", "NEW JERSEY": "NJ", "NEW MEXICO": "NM", "NEW YORK": "NY",
		"NORTH CAROLINA": "NC", "NORTH DAKOTA": "ND", "OHIO": "OH", "OKLAHOMA": "OK",
		"OREGON": "OR", "PENNSYLVANIA": "PA", "PUERTO RICO": "PR", "RHODE ISLAND": "RI",
		"SOUTH CAROLINA": "SC", "SOUTH DAKOTA": "SD", "TENNESSEE": "TN", "TEXAS": "TX",
		"UTAH": "UT", "VERMONT": "VT", "VIRGINIA": "VA", "WASHINGTON": "WA",
		"WEST VIRGINIA": "WV", "WISCONSIN": "WI", "WYOMING": "WY",
		"GUAM": "GU", "AMERICAN SAMOA": "AS", "VIRGIN ISLANDS": "VI",
	},
	"CA": {
		"ALBERTA": "AB", "BRITISH COLUMBIA": "BC", "MANITOBA": "MB",
		"NEW BRUNSWICK": "NB", "NEWFOUNDLAND AND LABRADOR": "NL", "NEWFOUNDLAND": "NL",
		"NOVA SCOTIA": "NS", "NORTHWEST TERRITORIES": "NT", "NUNAVUT": "NU",
		"ONTARIO": "ON", "PRINCE EDWARD ISLAND": "PE", "QUEBEC": "QC",
		"SASKATCHEWAN": "SK", "YUKON": "YT",
	},
	"AU": {
		"AUSTRALIAN CAPITAL TERRITORY": "ACT", "NEW SOUTH WALES": "NSW",
		"NORTHERN TERRITORY": "NT", "QUEENSLAND": "QLD", "SOUTH AUSTRALIA": "SA",
		"TASMANIA": "TAS", "VICTORIA": "VIC", "WESTERN AUSTRALIA": "WA",
	},
}

// countryZones is the zone for a country that keeps one, and the majority zone for the few multi-zone countries whose subdivisions are not enumerated above. Consulted only after subdivisionZones, so it doubles as the fallback for an unrecognised state.
var countryZones = map[string]string{
	// North and Central America
	"US": "America/New_York", "CA": "America/Toronto", "MX": "America/Mexico_City",
	"BZ": "America/Belize", "CR": "America/Costa_Rica", "SV": "America/El_Salvador",
	"GT": "America/Guatemala", "HN": "America/Tegucigalpa", "NI": "America/Managua",
	"PA": "America/Panama", "BS": "America/Nassau", "BB": "America/Barbados",
	"CU": "America/Havana", "DO": "America/Santo_Domingo", "HT": "America/Port-au-Prince",
	"JM": "America/Jamaica", "TT": "America/Port_of_Spain",
	// South America
	"AR": "America/Argentina/Buenos_Aires", "BO": "America/La_Paz", "BR": "America/Sao_Paulo",
	"CL": "America/Santiago", "CO": "America/Bogota", "EC": "America/Guayaquil",
	"GY": "America/Guyana", "PE": "America/Lima", "PY": "America/Asuncion",
	"SR": "America/Paramaribo", "UY": "America/Montevideo", "VE": "America/Caracas",
	// Europe
	"AL": "Europe/Tirane", "AT": "Europe/Vienna", "BA": "Europe/Sarajevo",
	"BE": "Europe/Brussels", "BG": "Europe/Sofia", "BY": "Europe/Minsk",
	"CH": "Europe/Zurich", "CY": "Asia/Nicosia", "CZ": "Europe/Prague",
	"DE": "Europe/Berlin", "DK": "Europe/Copenhagen", "EE": "Europe/Tallinn",
	"ES": "Europe/Madrid", "FI": "Europe/Helsinki", "FR": "Europe/Paris",
	"GB": "Europe/London", "GR": "Europe/Athens", "HR": "Europe/Zagreb",
	"HU": "Europe/Budapest", "IE": "Europe/Dublin", "IS": "Atlantic/Reykjavik",
	"IT": "Europe/Rome", "LT": "Europe/Vilnius", "LU": "Europe/Luxembourg",
	"LV": "Europe/Riga", "MD": "Europe/Chisinau", "ME": "Europe/Podgorica",
	"MK": "Europe/Skopje", "MT": "Europe/Malta", "NL": "Europe/Amsterdam",
	"NO": "Europe/Oslo", "PL": "Europe/Warsaw", "PT": "Europe/Lisbon",
	"RO": "Europe/Bucharest", "RS": "Europe/Belgrade", "RU": "Europe/Moscow",
	"SE": "Europe/Stockholm", "SI": "Europe/Ljubljana", "SK": "Europe/Bratislava",
	"TR": "Europe/Istanbul", "UA": "Europe/Kyiv",
	// Middle East and Africa
	"AE": "Asia/Dubai", "BH": "Asia/Bahrain", "DZ": "Africa/Algiers",
	"EG": "Africa/Cairo", "ET": "Africa/Addis_Ababa", "GH": "Africa/Accra",
	"IL": "Asia/Jerusalem", "IQ": "Asia/Baghdad", "IR": "Asia/Tehran",
	"JO": "Asia/Amman", "KE": "Africa/Nairobi", "KW": "Asia/Kuwait",
	"LB": "Asia/Beirut", "LY": "Africa/Tripoli", "MA": "Africa/Casablanca",
	"NG": "Africa/Lagos", "OM": "Asia/Muscat", "QA": "Asia/Qatar",
	"SA": "Asia/Riyadh", "SN": "Africa/Dakar", "TN": "Africa/Tunis",
	"TZ": "Africa/Dar_es_Salaam", "UG": "Africa/Kampala", "ZA": "Africa/Johannesburg",
	"ZW": "Africa/Harare",
	// Asia and Oceania
	"AU": "Australia/Sydney", "BD": "Asia/Dhaka", "CN": "Asia/Shanghai",
	"HK": "Asia/Hong_Kong", "ID": "Asia/Jakarta", "IN": "Asia/Kolkata",
	"JP": "Asia/Tokyo", "KH": "Asia/Phnom_Penh", "KR": "Asia/Seoul",
	"LA": "Asia/Vientiane", "LK": "Asia/Colombo", "MM": "Asia/Yangon",
	"MO": "Asia/Macau", "MY": "Asia/Kuala_Lumpur", "NP": "Asia/Kathmandu",
	"NZ": "Pacific/Auckland", "PH": "Asia/Manila", "PK": "Asia/Karachi",
	"SG": "Asia/Singapore", "TH": "Asia/Bangkok", "TW": "Asia/Taipei",
	"VN": "Asia/Ho_Chi_Minh",
}
