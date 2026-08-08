package loop

import (
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
)

// Execution pairs a tool call with its position in the model's emission and
// what it declared it would touch.
type Execution struct {
	// Index is the position the model emitted it at. Results are appended in
	// this order regardless of which finishes first.
	Index   int
	Call    ce.ToolCall
	Declare policy.Request
	// Err is a declaration failure; such a call is scheduled alone so its
	// error reaches the model in the right position.
	Err error
}

// Group is a set of executions that may run concurrently.
type Group []Execution

// Schedule splits calls into groups that are safe to run together.
//
// Pure, and deliberately so: scheduling is the easiest part to get subtly wrong
// and the cheapest to test in isolation.
//
// Two calls are separated when running them together would make the outcome
// depend on a race:
//
//   - two writes to the same path;
//   - a read and a write of the same path;
//   - any two system commands, whose side effects are arbitrary;
//   - anything that needs approval, since a user decision is sequential.
//
// Parallelism is kept for reads and searches, which is where it is both safe
// and where the wall-clock gain actually is.
func Schedule(execs []Execution, maxParallel int) []Group {
	if maxParallel <= 0 {
		maxParallel = 1
	}

	var groups []Group
	var current Group

	flush := func() {
		if len(current) > 0 {
			groups = append(groups, current)
			current = nil
		}
	}

	for _, e := range execs {
		if mustRunAlone(e) {
			flush()
			groups = append(groups, Group{e})
			continue
		}
		if len(current) >= maxParallel || conflictsWithAny(e, current) {
			flush()
		}
		current = append(current, e)
	}
	flush()
	return groups
}

// mustRunAlone reports calls that can never share a group.
func mustRunAlone(e Execution) bool {
	// A declaration failure carries no information about what it would touch,
	// so nothing can be proven safe alongside it.
	if e.Err != nil {
		return true
	}
	// A shell command is opaque: it could touch anything.
	if e.Declare.Command != "" {
		return true
	}
	// A write with no declared path is equally unbounded.
	for _, a := range e.Declare.Paths {
		if a.Write && a.Path == "" {
			return true
		}
	}
	return false
}

func conflictsWithAny(e Execution, group Group) bool {
	for _, other := range group {
		if conflicts(e, other) {
			return true
		}
	}
	return false
}

// conflicts reports whether two calls touch the same path in a way that makes
// concurrent execution racy.
func conflicts(a, b Execution) bool {
	for _, pa := range a.Declare.Paths {
		for _, pb := range b.Declare.Paths {
			if pa.Path != pb.Path {
				continue
			}
			// Two reads of the same path are fine; anything involving a write
			// is not.
			if pa.Write || pb.Write {
				return true
			}
		}
	}
	return false
}
