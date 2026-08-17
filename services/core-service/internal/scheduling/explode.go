package scheduling

import "sort"

// MaxExplosionDepth bounds the walk downstream of the constraint. Production graphs contain rework loops, so an unbounded walk would not terminate.
const MaxExplosionDepth = 10

// StepEdge is one directed link in the production-step graph, oriented upstream → downstream. The storage table spells this (A=downstream, B=upstream); the confusion stops at the query, and everything in here is already the right way round.
type StepEdge struct {
	UpstreamStepID   string
	DownstreamStepID string
}

// StepInfo is what the explosion needs to know about one production step.
type StepInfo struct {
	StepID       string
	DepartmentID string
	Name         string
	// LeadTimeOffsetWeeks is how far after the constraint campaign this step runs. It is a whole number of weeks because a schedule is planned in weeks; sub-week precision here would imply a resolution the plan does not have.
	LeadTimeOffsetWeeks int
	// YieldRatio is units of this step's output per unit of its input. 1 means no gain or loss; 0.95 means a 5% loss at this step.
	YieldRatio float64
}

// ExplosionInput is everything needed to derive department work from a constraint plan.
type ExplosionInput struct {
	// Campaigns are the constraint-level lines the plan already committed to.
	Campaigns []ExplosionCampaign
	Edges     []StepEdge
	Steps     map[string]StepInfo
	MaxDepth  int
}

// ExplosionCampaign is one constraint campaign to explode downstream.
type ExplosionCampaign struct {
	LineID    string
	ItemID    string
	SKU       string
	MachineID string
	WeekIndex int
	Quantity  float64
	// StepID is where the campaign runs — the constraint step the walk starts from.
	StepID string
}

// DerivedLine is work a constraint campaign implies for a department — the constraint step's own work at depth 0, and everything downstream of it below that.
type DerivedLine struct {
	SourceLineID   string
	ProductionStep string
	DepartmentID   string
	ItemID         string
	SKU            string
	// WeekIndex is the constraint week plus the accumulated lead-time offset.
	WeekIndex int
	Quantity  float64
	// Depth is how many steps downstream of the constraint this work sits, which is what a readiness chip keys off. Zero is the constraint step itself.
	Depth int
}

// Explode walks the production-step graph from each constraint campaign and returns the work each department has to do as a result.
//
// Two properties matter more than the traversal itself:
//
// Determinism. Go randomizes map iteration, so every collection is sorted before it is walked. Without that, two runs over the same plan produce the same rows in a different order, and a diff between versions becomes unreadable.
//
// Termination. Production graphs contain rework loops — a quality step feeding back into the step before it — so the walk is bounded by depth and refuses to revisit a step it has already reached at the same depth. An unbounded walk on a real graph does not return.
func Explode(in ExplosionInput) []DerivedLine {
	maxDepth := in.MaxDepth
	if maxDepth <= 0 {
		maxDepth = MaxExplosionDepth
	}

	downstream := buildDownstreamIndex(in.Edges)

	campaigns := append([]ExplosionCampaign(nil), in.Campaigns...)
	sort.Slice(campaigns, func(i, j int) bool {
		if campaigns[i].WeekIndex != campaigns[j].WeekIndex {
			return campaigns[i].WeekIndex < campaigns[j].WeekIndex
		}
		if campaigns[i].SKU != campaigns[j].SKU {
			return campaigns[i].SKU < campaigns[j].SKU
		}
		return campaigns[i].LineID < campaigns[j].LineID
	})

	var out []DerivedLine
	for _, campaign := range campaigns {
		out = append(out, explodeOne(campaign, downstream, in.Steps, maxDepth)...)
	}
	return out
}

// buildDownstreamIndex maps each step to its downstream steps, sorted so the walk is reproducible.
func buildDownstreamIndex(edges []StepEdge) map[string][]string {
	index := map[string][]string{}
	for _, edge := range edges {
		index[edge.UpstreamStepID] = append(index[edge.UpstreamStepID], edge.DownstreamStepID)
	}
	for step := range index {
		sort.Strings(index[step])
	}
	return index
}

// walkState is one position in the downstream walk.
type walkState struct {
	stepID   string
	depth    int
	week     int
	quantity float64
}

func explodeOne(
	campaign ExplosionCampaign,
	downstream map[string][]string,
	steps map[string]StepInfo,
	maxDepth int,
) []DerivedLine {
	if campaign.StepID == "" {
		return nil
	}

	// Keyed by (step, depth) rather than by step alone: a rework loop legitimately revisits a step, and collapsing on step would drop the second pass entirely.
	visited := map[walkKey]bool{}

	queue := []walkState{{
		stepID:   campaign.StepID,
		depth:    0,
		week:     campaign.WeekIndex,
		quantity: campaign.Quantity,
	}}

	var out []DerivedLine

	// The constraint's own work is work. Emitting only what follows it meant a plant with nothing configured downstream of its constraint got an empty work list while its machines were fully booked — the plan was there, the page just refused to say so. The campaign quantity is already this step's output, so no yield applies to it.
	if step, ok := steps[campaign.StepID]; ok {
		out = append(out, DerivedLine{
			SourceLineID:   campaign.LineID,
			ProductionStep: campaign.StepID,
			DepartmentID:   step.DepartmentID,
			ItemID:         campaign.ItemID,
			SKU:            campaign.SKU,
			WeekIndex:      campaign.WeekIndex,
			Quantity:       campaign.Quantity,
			Depth:          0,
		})
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		for _, nextStepID := range downstream[current.stepID] {
			key := walkKey{stepID: nextStepID, depth: current.depth + 1}
			if visited[key] {
				continue
			}
			visited[key] = true

			step, ok := steps[nextStepID]
			if !ok {
				// A step with no metadata cannot be scheduled against a department, and inventing one would put work on a department that never agreed to it.
				continue
			}

			yield := step.YieldRatio
			if yield <= 0 {
				yield = 1
			}

			next := walkState{
				stepID:   nextStepID,
				depth:    current.depth + 1,
				week:     current.week + step.LeadTimeOffsetWeeks,
				quantity: current.quantity * yield,
			}

			out = append(out, DerivedLine{
				SourceLineID:   campaign.LineID,
				ProductionStep: nextStepID,
				DepartmentID:   step.DepartmentID,
				ItemID:         campaign.ItemID,
				SKU:            campaign.SKU,
				WeekIndex:      next.week,
				Quantity:       next.quantity,
				Depth:          next.depth,
			})

			queue = append(queue, next)
		}
	}

	return out
}

type walkKey struct {
	stepID string
	depth  int
}
