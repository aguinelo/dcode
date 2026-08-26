// Package qualifier raises a definition of done when there is none to read.
//
// This is the deterministic half: running each proposed criterion against the
// repository as it stands, before any work, and classifying what came back.
// Deriving the proposal is a model's job and the operator's signature is a
// person's; neither is here yet.
//
// Spec: docs/specs/architecture/done-qualifier/202608261730-done-qualifier.*.spec.md
package qualifier

// Proposal is a candidate definition of done.
//
// It is a proposal and nothing else: no part of it reaches the loop before a
// signature. Nothing here is authority.
type Proposal struct {
	Criteria  []Proposed
	Protected []string
}

// Proposed is one candidate criterion.
type Proposed struct {
	// Name is what the report prints.
	Name string
	// Command is what decides. A criterion is a command, never a sentence.
	Command string
	// ExitCode is what counts as met; zero by default.
	ExitCode int
	// Expects is what the PROPOSER says this will do against the repository as
	// it stands, before any work.
	//
	// It decides nothing — the class comes from the run. What it produces is
	// the DISAGREEMENT: said it would fail, and it passed. That is the exact
	// signature of a criterion that does not measure what it should, and
	// without it "criterion 2 passed" is a neutral fact.
	Expects Expectation
	// Why is one line for the human deciding. No machine consumes it.
	Why string
}

// Expectation is the proposer's own claim about the state before any work.
type Expectation string

const (
	// ExpectFail — an acceptance criterion. The work has not happened, so this
	// has to be red now.
	ExpectFail Expectation = "fail"
	// ExpectPass — a regression guard. It works today and must keep working.
	ExpectPass Expectation = "pass"
)
