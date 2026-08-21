package tui

import "strings"

// The activity line says what is happening while a turn runs. The verb is the
// part that moves; the tool and its target are the part that is true.
//
// The rule the design states and this file exists to hold: **the verb never
// appears alone**. A gerund with nothing beside it is motion pretending to be
// information — the screen looks alive and the reader learns nothing. So a verb
// is only ever drawn next to a running tool, and when there is no tool the line
// says the one plain word it has always said.
//
// Everything here is a pure function of the frame counter, which is what keeps
// View pure and lets the whole thing be tested without a terminal.

// verbFrames is how many animation frames one verb holds.
//
// Twenty, because the tick is 120ms and the design asks for a change every
// 2.4 seconds. Written as frames rather than as a duration on purpose: the
// render reads a counter, never a clock, and two sources for one rhythm is how
// an animation drifts from the thing it animates.
const verbFrames = 20

// Phases of work, as a person watching would name them. The names are keys, not
// text: what the user sees comes from the catalogue below, in their language.
const (
	phaseReading    = "reading"
	phaseWriting    = "writing"
	phaseDelegating = "delegating"
	phaseRunning    = "running"
	phaseOther      = "other"
)

// activityPhase groups a tool by what is going on, not by what it is called.
//
// An unknown tool lands in phaseOther rather than in no phase at all: a tool
// added later should read as ordinary work, not disappear from the line.
func activityPhase(tool string) string {
	switch strings.ToLower(tool) {
	case "read", "glob", "grep", "symbol", "fetch":
		return phaseReading
	case "write", "edit":
		return phaseWriting
	case "explore":
		return phaseDelegating
	case "bash", "process":
		return phaseRunning
	default:
		return phaseOther
	}
}

// ActivityVerb returns the gerund to draw beside a running tool.
//
// Empty for an empty tool, and that is the invariant rather than an edge case:
// with no fact to accompany, there is nothing for a verb to be true about.
func ActivityVerb(tool string, frame int, l Lang) string {
	if tool == "" {
		return ""
	}
	set := activityVerbs[l]
	if set == nil {
		set = activityVerbs[Fallback]
	}
	verbs := set[activityPhase(tool)]
	if len(verbs) == 0 {
		return ""
	}
	if frame < 0 {
		frame = 0
	}
	return verbs[(frame/verbFrames)%len(verbs)]
}

// ActivityVerbsEnabled reads the one setting this costs.
//
// Default on. It is a client-side presentation choice and has nothing to do
// with `behavior.show_reasoning`: that one decides what the person is shown of
// the model's thinking, this one decides whether a line already on screen
// carries a word.
//
// Read here and passed in as geometry, because internal/tui never reads the
// environment — the same arrangement as the palette and the language.
func ActivityVerbsEnabled(env func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(env("DCODE_ACTIVITY_VERBS"))) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// activityVerbs is a catalogue of its own rather than fields on Strings.
//
// Strings is string-valued, and the coverage guard over it reads every field
// with reflect.Value.String() — a slice-valued field would come back non-empty
// whatever it held, and the guard would pass without checking anything. A new
// kind of entry gets a guard of its own instead of slipping past one written
// for something else.
var activityVerbs = map[Lang]map[string][]string{
	PtBR: {
		phaseReading:    {"lendo", "varrendo", "procurando"},
		phaseWriting:    {"escrevendo", "editando", "aplicando"},
		phaseDelegating: {"delegando", "repartindo", "coordenando"},
		phaseRunning:    {"rodando", "executando", "conferindo"},
		phaseOther:      {"processando", "organizando", "anotando"},
	},
	En: {
		phaseReading:    {"reading", "scanning", "searching"},
		phaseWriting:    {"writing", "editing", "applying"},
		phaseDelegating: {"delegating", "dividing", "coordinating"},
		phaseRunning:    {"running", "executing", "checking"},
		phaseOther:      {"processing", "organising", "noting"},
	},
}

// No verb repeats the word the line falls back to with no tool running. If the
// two were the same string, a reader could not tell a rotating verb from a
// still one — and neither could a test.

// activityPhases is every phase a tool can land in, for the guard.
func activityPhases() []string {
	return []string{phaseReading, phaseWriting, phaseDelegating, phaseRunning, phaseOther}
}
