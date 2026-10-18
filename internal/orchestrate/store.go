package orchestrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aguinelo/dcode/internal/protocol"
)

const (
	stateFile = "state.json"
	// stateVersion is State.V, the format of state.json.
	stateVersion = 1
)

// State is a project's state.json: what the daemon needs to conduct it again
// after a restart. The fields of later steps — the merge queue, the main's
// health — join it with the steps that fill them (.p §4).
type State struct {
	V       int              `json:"v"`
	Project protocol.Project `json:"project"`
	// Children are every child the project dispatched, terminal ones too:
	// /children hides a closed one after the retention, the state keeps it.
	Children  []protocol.Child    `json:"children"`
	Questions []protocol.Question `json:"questions"`
	// LogSeq is the seq of the last decision this state follows. The state on
	// disk is never ahead of the log (.p §8).
	LogSeq uint64 `json:"log_seq"`
	// Spent is the tokens of every session of the project — input, output and
	// cache read — in the ceiling's window (.p §10).
	Spent int64 `json:"spent"`
	// PendingReports are the reports no orchestrator turn has carried yet:
	// the inbox, kept on disk so a report survives a restart (.p §10).
	PendingReports []string `json:"pending_reports,omitempty"`
	// Briefs are what each child was dispatched with, by child id: what the
	// pipeline needs after dispatch and the wire's Child does not carry.
	Briefs map[string]Brief `json:"briefs,omitempty"`
}

// Brief is a child's dispatch as the pipeline reads it later: the task the
// reviewer judges against, the pull request's text for the merge step, what
// the child last said of its work, and the red test it named.
type Brief struct {
	Task      string `json:"task"`
	PRTitle   string `json:"pr_title,omitempty"`
	PRBody    string `json:"pr_body,omitempty"`
	FixesMain bool   `json:"fixes_main,omitempty"`
	// Claim is the conclusion of the child's last turn: what it said, kept
	// beside the measurement and never read as one.
	Claim string `json:"claim,omitempty"`
	// RedTest is the command the child named with red_test in its red phase,
	// until the harness runs it.
	RedTest string `json:"red_test,omitempty"`
}

// Store is one project's state.json.
type Store struct {
	path string
}

func storeAt(dir, id string) *Store {
	return &Store{path: filepath.Join(dir, id, stateFile)}
}

// Load reads the state. One that does not read, of another version, or of
// another project is an error naming the file — never an empty state in its
// place, which would be a project that forgot its orchestrator.
func (s *Store) Load() (State, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return State{}, fmt.Errorf("orchestrate: read %s: %w", s.path, err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, fmt.Errorf("orchestrate: %s does not read: %w", s.path, err)
	}
	if st.V != stateVersion {
		return State{}, fmt.Errorf("orchestrate: %s is format v%d, this daemon knows v%d", s.path, st.V, stateVersion)
	}
	if want := filepath.Base(filepath.Dir(s.path)); st.Project.ID != want {
		return State{}, fmt.Errorf("orchestrate: %s holds project %q, not %s", s.path, st.Project.ID, want)
	}
	return st, nil
}

// Save rewrites the state atomically: a temporary file beside it, synced,
// renamed over it, and the directory synced. A crash leaves the old state or
// the new one, never half of either.
func (s *Store) Save(st State) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("orchestrate: encode %s: %w", s.path, err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("orchestrate: create %s: %w", dir, err)
	}
	f, err := os.CreateTemp(dir, stateFile+".*")
	if err != nil {
		return fmt.Errorf("orchestrate: save %s: %w", s.path, err)
	}
	tmp := f.Name()
	_, err = f.Write(append(data, '\n'))
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, s.path)
	}
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("orchestrate: save %s: %w", s.path, err)
	}
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("orchestrate: sync %s: %w", dir, err)
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("orchestrate: sync %s: %w", dir, err)
	}
	return nil
}
