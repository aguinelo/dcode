package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
)

// BackgroundRunner starts a command that outlives the call that started it.
//
// Separate from Runner rather than a method on it: a foreground command is
// waited on and a background one is not, and the two return different things.
// Folding them together would give one method two meanings decided by a flag,
// which is the shape Declare exists to keep out of this package.
type BackgroundRunner interface {
	Start(ctx context.Context, workdir, command string) (Handle, error)
}

// Handle is a command that is still running, or one that has finished and whose
// output nobody has collected yet.
type Handle interface {
	// Output is everything written so far, stdout and stderr together.
	Output() string
	// Exited reports the exit code, and whether it has exited at all.
	Exited() (code int, done bool)
	// Stop ends it. Calling it on a command that already exited does nothing.
	Stop()
}

// procEntry is one background command and what started it.
type procEntry struct {
	id      string
	command string
	handle  Handle
	stopped bool
}

// AddProcess records a started command and returns the identifier the model
// will use to reach it again.
//
// The identifier is a sequence, not a clock. A timestamp would ride into tool
// output and make two otherwise identical sessions differ byte-for-byte, which
// is the same reason MarkRead takes a message index instead of a time.
func (s *State) AddProcess(command string, h Handle) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("bg%d", len(s.procs)+1)
	s.procs = append(s.procs, &procEntry{id: id, command: command, handle: h})
	return id
}

// process finds one entry by identifier.
func (s *State) process(id string) (*procEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.procs {
		if p.id == id {
			return p, true
		}
	}
	return nil, false
}

// processIDs lists every identifier handed out, in the order they were.
func (s *State) processIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.procs))
	for _, p := range s.procs {
		ids = append(ids, p.id)
	}
	return ids
}

// Close stops every background process this session started.
//
// This is where "a process dies with the session" stops being a rule someone
// has to remember and becomes ownership: the table lives in session state, and
// state that goes away takes the processes with it. It is also what makes
// approving a long command need no separate question about duration — the
// authorisation and the process have the same lifetime, so there is no window
// in which someone consented to an instant and granted an era.
func (s *State) Close() {
	s.mu.Lock()
	procs := make([]*procEntry, len(s.procs))
	copy(procs, s.procs)
	s.mu.Unlock()

	for _, p := range procs {
		p.handle.Stop()
	}
}

// ---------- the process tool ----------

// Process reads and stops the commands bash started in the background.
//
// A separate tool from bash, and the reason is the policy verdict rather than
// the tool count. bash declares the network and a write to the workspace,
// because a shell command is opaque and the worst case is what gets declared.
// Reading a buffer this process already owns crosses nothing at all. Folded
// together, every read of a log would queue an approval for a boundary it does
// not touch, and a question asked for no reason is how people learn to answer
// without reading.
type Process struct{}

// ProcessInput is the argument shape.
type ProcessInput struct {
	ID   string `json:"id,omitempty"`
	Stop bool   `json:"stop,omitempty"`
}

func (Process) Name() string { return "process" }

func (Process) Description() string {
	return "Read or stop a command started with `bash` in the background. " +
		"Give the identifier to see what it has printed since it started, " +
		"or call it with no identifier to list what is running. " +
		"Set stop to end one — do that when a server has served its purpose, " +
		"or before starting another on the same port. " +
		"Everything started this way ends when the session does."
}

func (Process) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"id":{"type":"string","description":"The identifier bash returned. Omit to list every process."},` +
		`"stop":{"type":"boolean","description":"End the process instead of only reading it."}}}`)
}

// Declare reports nothing, because nothing is touched: no path, no network, no
// command run. The verdict is always allow.
func (p Process) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: p.Name()}, nil
}

func (p Process) Execute(_ context.Context, input json.RawMessage, s *State) (Result, error) {
	var in ProcessInput
	if err := decode(p.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}

	if strings.TrimSpace(in.ID) == "" {
		return p.list(s), nil
	}

	entry, ok := s.process(in.ID)
	if !ok {
		known := s.processIDs()
		if len(known) == 0 {
			return errf(p.Name(), CodeBadInput,
				"Start one with `bash` and background set, then read it here.",
				"no background process has been started in this session").Result(), nil
		}
		return errf(p.Name(), CodeBadInput,
			fmt.Sprintf("Running: %s.", strings.Join(known, ", ")),
			"there is no background process called %q", in.ID).Result(), nil
	}

	if in.Stop {
		entry.handle.Stop()
		entry.stopped = true
	}
	return p.report(entry, s.Limits.MaxToolOutput), nil
}

// list is the answer to "what did I start".
func (p Process) list(s *State) Result {
	s.mu.Lock()
	procs := make([]*procEntry, len(s.procs))
	copy(procs, s.procs)
	s.mu.Unlock()

	if len(procs) == 0 {
		return Result{Output: "no background processes are running"}
	}
	var b strings.Builder
	for i, e := range procs {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s · %s · %s", e.id, status(e), e.command)
	}
	return Result{Output: b.String(), Meta: Meta{Files: len(procs)}}
}

// report is one process: what state it is in, and what it has printed.
func (p Process) report(e *procEntry, max int) Result {
	out, cut := clampTail(e.handle.Output(), max)
	body := out
	if strings.TrimSpace(body) == "" {
		body = "(no output)"
	}
	code, done := e.handle.Exited()
	res := Result{
		Output:    fmt.Sprintf("%s · %s · %s\n\n%s", e.id, status(e), e.command, body),
		Truncated: cut,
		Meta:      Meta{Lines: countLines(out)},
	}
	if done {
		res.Meta.ExitCode, res.Meta.HasExit = code, true
	}
	return res
}

// status is the one word that separates a live process from a finished one.
//
// The distinction matters more than it looks: a model reading output from a
// process it believes is running will wait for more, and a model reading a
// finished one has everything there will ever be.
func status(e *procEntry) string {
	code, done := e.handle.Exited()
	switch {
	case !done:
		return "running"
	case e.stopped:
		return "stopped"
	default:
		return fmt.Sprintf("exit %d", code)
	}
}

// clampTail keeps the end of the output rather than the beginning.
//
// The opposite of clamp, and deliberately: a server's first lines are a banner,
// and its last lines are why it died. Cutting the tail throws away the only
// part anyone reads a log for.
func clampTail(s string, max int) (string, bool) {
	if max <= 0 || len(s) <= max {
		return s, false
	}
	cut := s[len(s)-max:]
	// Start at a line boundary so the first line is not half a line.
	if i := strings.IndexByte(cut, '\n'); i >= 0 && i < len(cut)-1 {
		cut = cut[i+1:]
	}
	return fmt.Sprintf("… earlier output dropped, keeping the last %d bytes\n%s", max, cut), true
}
