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
	// Explore answers one question by reading. The implementation fixes the
	// child in read-only mode and excludes this tool from its registry — the
	// caller never chooses either.
	Explore(ctx context.Context, task, path string) (conclusion string, read, unread []string, truncated bool, err error)
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
		`"path":{"type":"string","description":"Directory to look under. Optional."}},` +
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
	// Declares a read and nothing else. The child cannot do more than this
	// whatever it is asked, because its mode says so.
	return policy.Request{Tool: e.Name(), Paths: []policy.Access{{Path: p}}}, nil
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
	if e.Delegator == nil {
		return errf(e.Name(), CodeNotFound,
			"Delegation is switched off in this session.",
			"no delegator is configured").Result(), nil
	}

	conclusion, read, unread, truncated, err := e.Delegator.Explore(ctx, in.Task, in.Path)
	if err != nil {
		return errf(e.Name(), CodeNotFound, "", "the delegated turn failed: %v", err).Result(), nil
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
