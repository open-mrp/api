package constants

// OeeBucket is the OEE term a stoppage charges.
//
// OeeBucketNotScheduled is the odd one out: it is removed from the Availability denominator entirely rather than counted as a loss against it, because a machine nobody planned to run has no OEE rather than 0% OEE.
type OeeBucket string

const (
	// OeeBucketAvailability indicates lost run time.
	OeeBucketAvailability OeeBucket = "availability"
	// OeeBucketPerformance indicates minor stops and speed loss.
	OeeBucketPerformance OeeBucket = "performance"
	// OeeBucketQuality indicates rework and holds.
	OeeBucketQuality OeeBucket = "quality"
	// OeeBucketNotScheduled indicates time the machine was never expected to run.
	OeeBucketNotScheduled OeeBucket = "not_scheduled"
)

func (b OeeBucket) IsValid() bool {
	switch b {
	case OeeBucketAvailability, OeeBucketPerformance, OeeBucketQuality, OeeBucketNotScheduled:
		return true
	default:
		return false
	}
}

func (b OeeBucket) EnumValues() []string {
	return []string{
		string(OeeBucketAvailability),
		string(OeeBucketPerformance),
		string(OeeBucketQuality),
		string(OeeBucketNotScheduled),
	}
}

// MachineDowntimeSource records how a stoppage came to be recorded.
type MachineDowntimeSource string

const (
	// MachineDowntimeSourceManual indicates a person logged the stoppage.
	MachineDowntimeSourceManual MachineDowntimeSource = "manual"
	// MachineDowntimeSourceScanner indicates a shop-floor station logged the stoppage.
	MachineDowntimeSourceScanner MachineDowntimeSource = "scanner"
	// MachineDowntimeSourceInferred indicates the system derived the stoppage from a gap in activity.
	MachineDowntimeSourceInferred MachineDowntimeSource = "inferred"
	// MachineDowntimeSourceAPI indicates an integration reported the stoppage.
	MachineDowntimeSourceAPI MachineDowntimeSource = "api"
)

func (s MachineDowntimeSource) IsValid() bool {
	switch s {
	case MachineDowntimeSourceManual, MachineDowntimeSourceScanner,
		MachineDowntimeSourceInferred, MachineDowntimeSourceAPI:
		return true
	default:
		return false
	}
}

func (s MachineDowntimeSource) EnumValues() []string {
	return []string{
		string(MachineDowntimeSourceManual),
		string(MachineDowntimeSourceScanner),
		string(MachineDowntimeSourceInferred),
		string(MachineDowntimeSourceAPI),
	}
}

// DowntimePlanningStatus is whether a stoppage was scheduled in advance. Planned maintenance and an unplanned breakdown are read very differently, so the distinction is named rather than left as a bare flag.
type DowntimePlanningStatus string

const (
	// DowntimePlanningStatusPlanned indicates the stoppage was scheduled in advance.
	DowntimePlanningStatusPlanned DowntimePlanningStatus = "planned"
	// DowntimePlanningStatusUnplanned indicates the stoppage was not scheduled.
	DowntimePlanningStatusUnplanned DowntimePlanningStatus = "unplanned"
)

func (s DowntimePlanningStatus) IsValid() bool {
	switch s {
	case DowntimePlanningStatusPlanned, DowntimePlanningStatusUnplanned:
		return true
	default:
		return false
	}
}

func (s DowntimePlanningStatus) EnumValues() []string {
	return []string{string(DowntimePlanningStatusPlanned), string(DowntimePlanningStatusUnplanned)}
}

// DowntimePlanningStatusOf maps the stored flag onto the enum the API exposes.
func DowntimePlanningStatusOf(planned bool) DowntimePlanningStatus {
	if planned {
		return DowntimePlanningStatusPlanned
	}
	return DowntimePlanningStatusUnplanned
}

// DowntimeStatus is whether a machine is still down.
type DowntimeStatus string

const (
	// DowntimeStatusOpen indicates the machine is down right now.
	DowntimeStatusOpen DowntimeStatus = "open"
	// DowntimeStatusClosed indicates the machine is running again.
	DowntimeStatusClosed DowntimeStatus = "closed"
)

func (s DowntimeStatus) IsValid() bool {
	switch s {
	case DowntimeStatusOpen, DowntimeStatusClosed:
		return true
	default:
		return false
	}
}

func (s DowntimeStatus) EnumValues() []string {
	return []string{string(DowntimeStatusOpen), string(DowntimeStatusClosed)}
}

// OeeMeasurementStatus says whether a grouping's OEE was measured from logged downtime or estimated from runtime.
//
// This matters more than it looks: a department with no logged downtime computes Availability as 100%, so its OEE jumps the day downtime logging ships. The status makes an estimate visibly an estimate instead of a suspiciously good measurement.
type OeeMeasurementStatus string

const (
	// OeeMeasurementStatusMeasured indicates availability came from logged downtime events.
	OeeMeasurementStatusMeasured OeeMeasurementStatus = "measured"
	// OeeMeasurementStatusEstimated indicates no downtime was logged, so availability was inferred.
	OeeMeasurementStatusEstimated OeeMeasurementStatus = "estimated"
)

func (s OeeMeasurementStatus) IsValid() bool {
	switch s {
	case OeeMeasurementStatusMeasured, OeeMeasurementStatusEstimated:
		return true
	default:
		return false
	}
}

func (s OeeMeasurementStatus) EnumValues() []string {
	return []string{string(OeeMeasurementStatusMeasured), string(OeeMeasurementStatusEstimated)}
}

func (b *OeeBucket) StringPtr() *string {
	if b == nil {
		return nil
	}
	str := string(*b)
	return &str
}

func (s *MachineDowntimeSource) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (s *DowntimePlanningStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (s *DowntimeStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

func (s *OeeMeasurementStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

// OeeAnomaly is a data-quality warning attached to an OEE result.
//
// Modelled as a list of named anomalies rather than one boolean per condition so a new warning does not add another flag to every department in every response.
type OeeAnomaly string

const (
	// OeeAnomalyPerformanceAboveCapacity indicates performance exceeded 100%, which means a stale ideal cycle time rather than a real result. The raw value is still reported; clamping it would hide the data problem.
	OeeAnomalyPerformanceAboveCapacity OeeAnomaly = "performance_above_capacity"
)

func (a OeeAnomaly) IsValid() bool {
	switch a {
	case OeeAnomalyPerformanceAboveCapacity:
		return true
	default:
		return false
	}
}

func (a OeeAnomaly) EnumValues() []string {
	return []string{string(OeeAnomalyPerformanceAboveCapacity)}
}

func (a *OeeAnomaly) StringPtr() *string {
	if a == nil {
		return nil
	}
	str := string(*a)
	return &str
}
