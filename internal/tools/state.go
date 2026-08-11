package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// State is the per-session state tools share.
type State struct {
	Resolver *policy.Resolver
	Limits   Limits

	mu    sync.Mutex
	files map[string]fileState
	plan  []protocol.PlanItem
}

type fileState struct {
	hash   string
	readAt int
}

// Limits are the per-tool caps.
type Limits struct {
	ReadMaxLines      int
	ReadMaxLineLength int
	GlobMaxResults    int
	GrepMaxMatches    int
	SymbolMaxMatches  int
	MaxToolOutput     int
	RespectGitignore  bool
	RequireReadBefore bool
	AtomicWrite       bool
	// EditEchoDiff decides when the diff of an edit goes back to the MODEL.
	// It always goes to the client, in every mode.
	EditEchoDiff string
}

// DefaultLimits mirrors the documented defaults.
func DefaultLimits() Limits {
	return Limits{
		ReadMaxLines:      2000,
		ReadMaxLineLength: 2000,
		GlobMaxResults:    1000,
		GrepMaxMatches:    200,
		SymbolMaxMatches:  200,
		MaxToolOutput:     256 * 1024,
		RespectGitignore:  true,
		RequireReadBefore: true,
		AtomicWrite:       true,
		EditEchoDiff:      EchoDiffMulti,
	}
}

// NewState builds session state.
func NewState(r *policy.Resolver, l Limits) *State {
	return &State{Resolver: r, Limits: l, files: map[string]fileState{}}
}

// MarkRead records that path was read with this content.
//
// msgIdx is the position in history, deliberately not a timestamp: a clock
// value here would leak into tool output and invalidate the prompt cache on
// every call.
func (s *State) MarkRead(path, content string, msgIdx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[path] = fileState{hash: hashOf(content), readAt: msgIdx}
}

// CheckEditable enforces the read-before-edit invariant.
//
// This is the rule that prevents the most expensive failure the product can
// produce: blind-editing a file whose contents the model assumed. Without it
// the agent silently overwrites work the user did in parallel.
func (s *State) CheckEditable(path, current string) *ToolError {
	if !s.Limits.RequireReadBefore {
		return nil
	}
	s.mu.Lock()
	st, seen := s.files[path]
	s.mu.Unlock()

	if !seen {
		return &ToolError{
			Code:   CodeFileNotRead,
			Reason: "this file has not been read in this session",
			Detail: path,
			Hint:   "Read it first, then edit. Editing from an assumed content is how files get corrupted.",
		}
	}
	if st.hash != hashOf(current) {
		return &ToolError{
			Code:   CodeFileChanged,
			Reason: "the file changed since it was read",
			Detail: path,
			Hint:   "Read it again to see the current content, then edit.",
		}
	}
	return nil
}

// WasRead reports whether path has been read this session.
func (s *State) WasRead(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.files[path]
	return ok
}

// Plan returns a copy of the current plan.
func (s *State) Plan() []protocol.PlanItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]protocol.PlanItem, len(s.plan))
	copy(out, s.plan)
	return out
}

// setPlan replaces the plan wholesale.
func (s *State) setPlan(items []protocol.PlanItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plan = items
}

func hashOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// ChangedSinceRead reports files whose content on disk no longer matches what
// the model was shown.
//
// The reader is injected so the check is testable without a filesystem, and so
// the caller decides what "on disk" means. A file that has become unreadable is
// not reported as changed: it is a different failure, and the tool that touches
// it next will say so precisely.
func (s *State) ChangedSinceRead(read func(path string) (string, error)) []string {
	s.mu.Lock()
	snapshot := make(map[string]string, len(s.files))
	for path, st := range s.files {
		snapshot[path] = st.hash
	}
	s.mu.Unlock()

	var changed []string
	for path, hash := range snapshot {
		current, err := read(path)
		if err != nil {
			continue
		}
		if hashOf(current) != hash {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return changed
}
