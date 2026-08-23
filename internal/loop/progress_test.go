package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

func progressOf(rec *recorder, kind string) []protocol.Progress {
	var out []protocol.Progress
	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, p := range rec.all {
		if pr, ok := p.(protocol.Progress); ok && pr.Kind == kind {
			out = append(out, pr)
		}
	}
	return out
}

// The turn says where it is against its ceiling, and the ceiling rides with the
// count. A count without its limit answers "how many" when the question is
// "how close", and a client carrying the limit separately would be carrying a
// copy of configuration it cannot see change.
func TestATurnReportsItsRoundAgainstItsCeiling(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"x"}`), done()},
		{text("done"), done()},
	}}
	e, rec := newEngine(t, p, reg)

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	got := progressOf(rec, protocol.ProgressRounds)
	if len(got) == 0 {
		t.Fatal("the turn never said which round it was on")
	}
	for i, pr := range got {
		if pr.Total != e.cfg.Limits.MaxIterations {
			t.Errorf("round %d carries ceiling %d, want %d", i, pr.Total, e.cfg.Limits.MaxIterations)
		}
		if pr.Done != i+1 {
			t.Errorf("round %d reported Done=%d", i+1, pr.Done)
		}
		if pr.ToolCallID != "" {
			t.Errorf("the turn's own progress carries a tool call id: %q", pr.ToolCallID)
		}
	}
}

// How many are about to run together, against what this session allows. It is
// emitted where the number exists — inside a group every call would report the
// same figure, and after it the number is already history.
func TestABatchReportsHowManyRunTogether(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a"}`), call("c2", "read", `{"path":"b"}`), done()},
		{text("done"), done()},
	}}
	e, rec := newEngine(t, p, reg)

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	got := progressOf(rec, protocol.ProgressInFlight)
	if len(got) == 0 {
		t.Fatal("a batch never said how many ran together")
	}
	for _, pr := range got {
		if pr.Total != e.cfg.Parallel {
			t.Errorf("in flight carries ceiling %d, want %d", pr.Total, e.cfg.Parallel)
		}
		if pr.Done < 1 || pr.Done > pr.Total {
			t.Errorf("in flight reported %d of %d", pr.Done, pr.Total)
		}
	}
}

// A turn that answered in one pass reports no round, and that is the contract
// rather than a gap.
//
// Iterations counts cycles the loop went round AGAIN — a turn that answered and
// passed the done-check returns without incrementing. There is no ceiling
// approaching, so there is no number to show, and emitting 0 of 100 would put a
// figure on screen that means nothing is happening.
func TestATurnThatAnsweredInOnePassReportsNoRound(t *testing.T) {
	reg := tools.NewRegistry()
	p := &scriptedProvider{turns: [][]provider.StreamEvent{{text("just an answer"), done()}}}
	e, rec := newEngine(t, p, reg)

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if got := progressOf(rec, protocol.ProgressRounds); len(got) != 0 {
		t.Errorf("a turn that never looped reported %d rounds: %+v", len(got), got)
	}
}

// Every kind emitted is one the protocol declares. A kind invented at the point
// of emission is a word on a versioned surface that nothing documents, and the
// client would render it as an unknown counter rather than as nothing.
func TestEveryKindEmittedIsOneTheProtocolDeclares(t *testing.T) {
	known := map[string]bool{
		protocol.ProgressRounds:   true,
		protocol.ProgressInFlight: true,
	}
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a"}`), done()},
		{text("done"), done()},
	}}
	e, rec := newEngine(t, p, reg)
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, pl := range rec.all {
		pr, ok := pl.(protocol.Progress)
		if !ok {
			continue
		}
		if !known[pr.Kind] {
			t.Errorf("emitted an undeclared kind %q", pr.Kind)
		}
	}
}

// Progress never reaches the model. It is a person's window — the same rule
// StartedAt already carries — and a count that differs between two runs of one
// session is exactly what ADR-03 forbids in a prefix.
func TestProgressNeverEntersTheContextSentToTheModel(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a"}`), done()},
		{text("done"), done()},
	}}
	e, _ := newEngine(t, p, reg)
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	for _, m := range e.Session().History {
		for _, word := range []string{protocol.ProgressRounds, protocol.ProgressInFlight} {
			if strings.Contains(m.Text, word) {
				t.Errorf("progress reached the model's history: %q", m.Text)
			}
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, req := range p.seen {
		for _, m := range req.Messages {
			if strings.Contains(m.Text, protocol.ProgressRounds) {
				t.Errorf("progress was sent to the provider: %q", m.Text)
			}
		}
	}
}

// A scan says how far it has got, and the reports name the call they came from.
//
// grep knows its list before it starts, so it can say `n of N`. glob is
// discovering as it walks, so it sends the count alone — a total it has not
// finished counting would be a number it made up.
func TestAScanReportsHowFarItHasGot(t *testing.T) {
	ws := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(ws, name+".go"), []byte("package x\n// needle\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reg := tools.NewRegistry(tools.Grep{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "grep", `{"pattern":"needle"}`), done()},
		{text("done"), done()},
	}}
	res, err := policy.NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	e, rec := newEngine(t, p, reg, func(c *Config) {
		c.State = tools.NewState(res, tools.DefaultLimits(), allToolNames)
	})
	if _, err := e.Run(context.Background(), "find it"); err != nil {
		t.Fatal(err)
	}

	got := progressOf(rec, protocol.ProgressFiles)
	if len(got) == 0 {
		t.Fatal("the scan never said how far it had got")
	}
	for _, pr := range got {
		if pr.ToolCallID != "c1" {
			t.Errorf("a scan's report does not name its call: %q", pr.ToolCallID)
		}
		if pr.Done < 1 {
			t.Errorf("reported %d done", pr.Done)
		}
		if pr.Total != 0 && pr.Done > pr.Total {
			t.Errorf("reported %d of %d", pr.Done, pr.Total)
		}
	}
}

// A tool that reports through the context reports for ITS call, and two running
// together do not write through one another. State is per session and shared,
// which is why the reporter is not on it.
func TestTwoScansDoNotReportThroughEachOther(t *testing.T) {
	seen := map[string][]int{}
	var mu sync.Mutex
	ctxOf := func(id string) context.Context {
		return tools.WithProgress(context.Background(), func(_ string, done, _ int) {
			mu.Lock()
			defer mu.Unlock()
			seen[id] = append(seen[id], done)
		})
	}
	tools.Progress(ctxOf("a"))("files", 1, 0)
	tools.Progress(ctxOf("b"))("files", 2, 0)

	if len(seen["a"]) != 1 || seen["a"][0] != 1 {
		t.Errorf("a got %v", seen["a"])
	}
	if len(seen["b"]) != 1 || seen["b"][0] != 2 {
		t.Errorf("b got %v", seen["b"])
	}
}

// A context with no reporter is not a crash. A tool should say how far it has
// got without first asking whether anybody is listening.
func TestAToolCanReportWithNobodyListening(t *testing.T) {
	tools.Progress(context.Background())("files", 1, 2)
	tools.Reporter(context.Background(), "files", 0)(1)
}

// glob is discovering as it walks, so it sends the count alone. A total it has
// not finished counting would be a number it made up, and a denominator that
// grows while a fraction is on screen is worse than no fraction.
func TestAWalkThatIsStillDiscoveringSendsNoTotal(t *testing.T) {
	ws := t.TempDir()
	for i := 0; i < 30; i++ {
		if err := os.WriteFile(filepath.Join(ws, fmt.Sprintf("f%02d.go", i)), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reg := tools.NewRegistry(tools.Glob{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "glob", `{"pattern":"**/*.go"}`), done()},
		{text("done"), done()},
	}}
	res, err := policy.NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	e, rec := newEngine(t, p, reg, func(c *Config) {
		c.State = tools.NewState(res, tools.DefaultLimits(), allToolNames)
	})
	if _, err := e.Run(context.Background(), "list them"); err != nil {
		t.Fatal(err)
	}

	got := progressOf(rec, protocol.ProgressFiles)
	if len(got) == 0 {
		t.Fatal("the walk never said how far it had got")
	}
	for _, pr := range got {
		if pr.Total != 0 {
			t.Errorf("a walk still discovering reported a total of %d", pr.Total)
		}
	}
}

// A call announces itself the moment its name is known, and reports its
// arguments as they arrive — throttled, because a fragment can be a handful of
// bytes and one event per fragment would put thousands of lines in the record
// of a single large write.
func TestACallAnnouncesItselfAndItsArgumentsAsTheyArrive(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	big := strings.Repeat("x", argumentStep*3)
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{
			{Type: provider.EventToolCallOpened, CallID: "c1", CallName: "read"},
			{Type: provider.EventToolCallProgress, CallID: "c1", CallName: "read", Bytes: 10},
			{Type: provider.EventToolCallProgress, CallID: "c1", CallName: "read", Bytes: argumentStep + 1},
			{Type: provider.EventToolCallProgress, CallID: "c1", CallName: "read", Bytes: len(big)},
			call("c1", "read", `{"path":"a"}`), done(),
		},
		{text("done"), done()},
	}}
	e, rec := newEngine(t, p, reg)
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	got := progressOf(rec, protocol.ProgressArguments)
	if len(got) == 0 {
		t.Fatal("the call never announced itself")
	}
	if got[0].Done != 0 || got[0].Name != "read" {
		t.Errorf("the first report does not open the line: %+v", got[0])
	}
	for _, pr := range got {
		if pr.ToolCallID != "c1" {
			t.Errorf("a report does not name its call: %+v", pr)
		}
		if pr.Total != 0 {
			t.Errorf("a total was invented for a call still arriving: %d", pr.Total)
		}
	}
	// The ten-byte fragment is below the step and must not have been sent.
	for _, pr := range got {
		if pr.Done == 10 {
			t.Error("a fragment below the step was reported")
		}
	}
}
