package service

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/open-mrp/api/services/agent-service/internal/agents"
	"github.com/open-mrp/api/shared/constants"
)

const (
	// doomLoopThreshold is how many times the same call may appear in the recent window before it is refused.
	doomLoopThreshold = 3

	// doomLoopWindowSize is the maximum number of recent tool calls tracked.
	doomLoopWindowSize = 20

	// noProgressSteerAfter is the run of consecutive no-progress iterations that earns one steering message.
	noProgressSteerAfter = 4

	// noProgressWrapUpAfter is the run of consecutive no-progress iterations that ends the turn early.
	noProgressWrapUpAfter = 6
)

// loopVerdict is what the runner should do after an iteration, judged by doomLoopDetector.EndIteration.
type loopVerdict int

const (
	loopContinue loopVerdict = iota
	loopSteer
	loopWrapUp
)

// noProgressSteeringPrompt is appended to the tool results once an agent has spent several steps only searching or failing.
const noProgressSteeringPrompt = "You have spent several steps searching or hitting errors without making progress. Stop searching for other ways to do this. If your tools don't offer the operation or data you need, tell the user plainly what is missing. Otherwise, answer with what you have."

// doomLoopDetector spots an agent that repeats the same call or stops making progress.
type doomLoopDetector struct {
	history    []string
	noProgress int
	steered    bool
}

// Record adds a tool call and reports whether the same call now appears doomLoopThreshold times in the recent window.
func (d *doomLoopDetector) Record(toolName string, input json.RawMessage) bool {
	fp := toolCallFingerprint(toolName, input)
	d.history = append(d.history, fp)
	if len(d.history) > doomLoopWindowSize {
		d.history = d.history[len(d.history)-doomLoopWindowSize:]
	}

	count := 0
	for _, h := range d.history {
		if h == fp {
			count++
		}
	}
	return count >= doomLoopThreshold
}

// EndIteration records whether an iteration made progress and says whether to steer the agent or stop the turn.
func (d *doomLoopDetector) EndIteration(progressed bool) loopVerdict {
	if progressed {
		d.noProgress = 0
		return loopContinue
	}
	d.noProgress++
	switch {
	case d.noProgress >= noProgressWrapUpAfter:
		return loopWrapUp
	case d.noProgress >= noProgressSteerAfter && !d.steered:
		d.steered = true
		return loopSteer
	default:
		return loopContinue
	}
}

// madeProgress reports whether a successful call of this tool counts as progress; discovery alone does not.
func madeProgress(toolName string) bool {
	return toolName != agents.SearchAPIToolsSlug && toolName != string(constants.ToolDescribeApiOperation)
}

// toolCallFingerprint identifies a call independent of JSON key order and whitespace. URL-reading tools are keyed
// on the normalized URL alone, so trivially different spellings of the same page count as repeats.
func toolCallFingerprint(toolName string, input json.RawMessage) string {
	switch constants.Tool(toolName) {
	case constants.ToolFetchUrl, constants.ToolReadDoc:
		var p struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(input, &p) == nil && p.URL != "" {
			return toolName + "\x00" + normalizeURL(p.URL)
		}
	}
	return toolCallApprovalKey(toolName, input)
}

// normalizeURL lowercases scheme and host and drops the fragment, a trailing slash, and a trailing ".md".
func normalizeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.Path = strings.TrimSuffix(strings.TrimSuffix(u.Path, "/"), ".md")
	return u.String()
}
