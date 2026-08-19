package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
)

// Delegator runs a read-only sub-turn and reports what it found.
//
// An interface for the same reason Bash takes a Runner: the loop owns turns and
// this package owns tools, and having either import the other would close a
// cycle. The tool declares and validates; the loop decides what a child turn is
// allowed to be.
type Delegator interface {
	// Explore answers one question, reading and — when owns is non-empty —
	// writing the paths it names.
	//
	// The implementation excludes this tool from the child's registry, so
	// nesting stays impossible by absence. It also decides the mode: read-only
	// when owns is empty, and otherwise the PARENT's mode, intersected with
	// the owned paths. The caller never chooses either, which is why owns is a
	// request rather than a grant.
	Explore(ctx context.Context, task, path string, owns []string) (conclusion string, read, wrote, unread []string, truncated bool, err error)
}

// Explore delegates reading, so the cost of it does not come back.
//
// dcode has one head. A task crossing twenty files reads all twenty in the same
// window, and each one pushes the last out; by file fifteen compaction hits and
// what was learned at file three becomes two lines of summary, if it survives at
// all.
//
// Delegating inverts that. A child reads the twenty files in ITS window and
// returns half a page. The gain is not speed — it is that the cost of the
// reading does not come back.
type Explore struct {
	Delegator Delegator
}

// ExploreInput is the argument shape.
//
// The child receives the TASK, not the parent's history. That is the whole
// point: a copied history would return exactly the cost delegation exists to
// avoid.
//
// There is no mode field, and that absence is the guarantee. Read-only is fixed
// where the sub-turn is constructed, so it is not something the model passes and
// not something a caller forgets.
type ExploreInput struct {
	Task string `json:"task"`
	Path string `json:"path,omitempty"`
	// Owns are the paths the child may write, and it is the only way a child
	// writes at all.
	//
	// Absent is the read-only child that already existed, so nothing changes
	// for anyone already delegating. Present is a request, never a grant: the
	// loop intersects it with what the parent may already write, and a child
	// can only ever end up narrower than its parent.
	//
	// There is still no mode field, and the absence is still the guarantee.
	// What the model passes is the task and the paths, and both may only
	// narrow.
	Owns []string `json:"owns,omitempty"`
}

func (Explore) Name() string { return "explore" }

func (Explore) Description() string {
	return "Delegate a read-only investigation to a fresh context, and get back a short answer. " +
		"Use it when answering would mean reading many files you will not need afterwards — " +
		"the files it reads do not consume your context, only its answer does. " +
		"It can only read: no editing, no commands, and it cannot delegate further. " +
		"Do not use it for something you have already read, or for a single known file: " +
		"a delegated turn costs more than one read."
}

func (Explore) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"task":{"type":"string","description":"The question, in one sentence."},` +
		`"path":{"type":"string","description":"Directory to look under. Optional."},` +
		`"owns":{"type":"array","items":{"type":"string"},"description":` +
		`"Paths the child may write. Omit for a read-only child. Naming a path here is a request, not a grant: it is narrowed to what you may already write. Two children must never own the same path."}},` +
		`"required":["task"]}`)
}

func (e Explore) Declare(input json.RawMessage) (policy.Request, error) {
	var in ExploreInput
	if err := decode(e.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	p := in.Path
	if p == "" {
		p = "."
	}
	// The read the child will do, and then every path it asked to own — as a
	// WRITE, because that is what the scheduler serialises on. Two children
	// owning one file have to be visible as a conflict before either runs;
	// ownership that never reached the declaration would be discovered by the
	// filesystem instead.
	paths := []policy.Access{{Path: p}}
	for _, own := range in.Owns {
		paths = append(paths, policy.Access{Path: own, Write: true})
	}
	return policy.Request{Tool: e.Name(), Paths: paths}, nil
}

func (e Explore) Execute(ctx context.Context, input json.RawMessage, s *State) (Result, error) {
	var in ExploreInput
	if err := decode(e.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if strings.TrimSpace(in.Task) == "" {
		return errf(e.Name(), CodeInvalidPattern,
			"Say what the child should find out, in one sentence.",
			"task is empty").Result(), nil
	}
	// An empty set is not "everything". It has the same standing a write with
	// no declared path already has: nothing said is never a yes.
	if in.Owns != nil && len(in.Owns) == 0 {
		return errf(e.Name(), CodeInvalidPattern,
			"Name the paths the child may write, or leave owns out for a child that only reads.",
			"owns is present and empty").Result(), nil
	}
	if e.Delegator == nil {
		return errf(e.Name(), CodeNotFound,
			"Delegation is switched off in this session.",
			"no delegator is configured").Result(), nil
	}

	conclusion, read, wrote, unread, truncated, err := e.Delegator.Explore(ctx, in.Task, in.Path, in.Owns)
	if err != nil {
		// Named, never summarised away. With several children in flight, one
		// that did not answer has to be identifiable — an incomplete result
		// wearing the face of a complete one is the defect this whole tool
		// exists to avoid producing.
		return errf(e.Name(), CodeNotFound, "",
			"the delegated turn failed: %v (task: %s)", err, in.Task).Result(), nil
	}

	var b strings.Builder
	b.WriteString(conclusion)
	if truncated {
		b.WriteString("\n\n… report truncated.")
	}
	// The paths always travel with the conclusion. They do not prove the child
	// understood; they prove it looked, which turns "trust me" into something
	// that can be spot-checked.
	if len(read) > 0 {
		b.WriteString("\n\nlooked at: " + strings.Join(read, ", "))
	}
	// What it changed, beside where it looked. A divided piece of work raises
	// "which child touched what" before it raises anything else.
	if len(wrote) > 0 {
		b.WriteString("\nwrote: " + strings.Join(wrote, ", "))
	}
	if len(unread) > 0 {
		// Never swallowed. A conclusion with an undeclared hole is a wrong
		// conclusion wearing the face of a complete one (RN-5, RN-10).
		b.WriteString("\ncould not read: " + strings.Join(unread, ", "))
	}

	return Result{
		Output:    b.String(),
		Truncated: truncated,
		Meta:      Meta{Files: len(read)},
	}, nil
}
