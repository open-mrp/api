package apiresource

import (
	"github.com/augno/api/shared/timeutil"
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

// SampleAnalyticsPeriodStart is the canonical analytics range start for OpenAPI request examples.
var SampleAnalyticsPeriodStart = timeutil.TimestampToTime(sampleCreatedAtTimestamp)

// SampleAnalyticsPeriodEnd is the canonical analytics range end for OpenAPI request examples.
var SampleAnalyticsPeriodEnd = timeutil.TimestampToTime(sampleUpdatedAtTimestamp)
