package apiresource

import (
	"github.com/open-mrp/api/shared/timeutil"
)

const sampleYear = "2026"
const sampleMonth = "05"
const sampleDay = "10"

const sampleCreatedAtTimestamp = sampleYear + "-" + sampleMonth + "-" + sampleDay + "T00:00:00Z"
const sampleUpdatedAtTimestamp = sampleYear + "-" + sampleMonth + "-" + sampleDay + "T00:23:00Z"
const sampleExpiresAtTimestamp = sampleYear + "-" + "06" + "-" + sampleDay + "T00:00:00Z"

// SampleFilterStartDateRFC3339 is used in OpenAPI examples for date-range query parameters.
const SampleFilterStartDateRFC3339 = sampleCreatedAtTimestamp

// SampleFilterEndDateRFC3339 is used in OpenAPI examples for date-range query parameters.
const SampleFilterEndDateRFC3339 = sampleUpdatedAtTimestamp

// SampleFilterDateOnly is used in OpenAPI examples for query parameters that take a plain calendar date rather than an instant.
const SampleFilterDateOnly = sampleYear + "-" + sampleMonth + "-" + sampleDay

// SampleAnalyticsPeriodStart is the canonical analytics range start for OpenAPI request examples.
var SampleAnalyticsPeriodStart = timeutil.TimestampToTime(sampleCreatedAtTimestamp)

// SampleAnalyticsPeriodEnd is the canonical analytics range end for OpenAPI request examples.
var SampleAnalyticsPeriodEnd = timeutil.TimestampToTime(sampleUpdatedAtTimestamp)
