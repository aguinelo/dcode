// Package config resolves where files live and what the effective settings are.
//
// One precedence chain for the whole product, no per-key special cases:
//
//	locked > flag > environment > project config > user config > default
//
// Every value keeps its origin, because configuration that cannot explain
// itself produces the worst kind of support conversation.
//
// Spec: docs/specs/architecture/configuration/202608081203-*.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// Source is where a value came from.
type Source string

const (
	SourceLocked  Source = "locked"
	SourceFlag    Source = "flag"
	SourceEnv     Source = "env"
	SourceProject Source = "project"
	SourceUser    Source = "user"
	SourceDefault Source = "default"
)

// rank orders the chain from strongest to weakest.
var rank = map[Source]int{
	SourceLocked: 6, SourceFlag: 5, SourceEnv: 4,
	SourceProject: 3, SourceUser: 2, SourceDefault: 1,
}

// Value is a resolved setting with its provenance.
type Value struct {
	Key    string
	Value  string
	Source Source
	Origin string
	Locked bool
}

// Layer is one source of settings.
type Layer struct {
	Source Source
	Origin string
	Values map[string]string
}

// Resolved is the outcome of the chain.
type Resolved struct {
	values map[string]Value
	// Warnings records attempts to override a locked key. Silently ignoring
	// them would leave the user believing a change took effect.
	Warnings []string
}

// Get returns a value and whether it was set.
func (r Resolved) Get(key string) (Value, bool) {
	v, ok := r.values[key]
	return v, ok
}

// String returns the effective value, or def.
func (r Resolved) String(key, def string) string {
	if v, ok := r.values[key]; ok && v.Value != "" {
		return v.Value
	}
	return def
}

// Bool returns the effective boolean.
func (r Resolved) Bool(key string, def bool) bool {
	v, ok := r.values[key]
	if !ok || v.Value == "" {
		return def
	}
	switch strings.ToLower(v.Value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

// Int returns the effective integer.
func (r Resolved) Int(key string, def int) int {
	v, ok := r.values[key]
	if !ok || v.Value == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v.Value, "%d", &n); err != nil {
		return def
	}
	return n
}

// Keys lists resolved keys, sorted.
func (r Resolved) Keys() []string {
	out := make([]string, 0, len(r.values))
	for k := range r.values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Resolve applies the chain. Pure over already-loaded layers.
func Resolve(layers []Layer) Resolved {
	out := Resolved{values: map[string]Value{}}
	locked := map[string]bool{}

	for _, l := range layers {
		if l.Source == SourceLocked {
			for k := range l.Values {
				locked[k] = true
			}
		}
	}

	for _, l := range layers {
		for k, v := range l.Values {
			if locked[k] && l.Source != SourceLocked {
				out.Warnings = append(out.Warnings, fmt.Sprintf(
					"%s is locked by the administrator; the value from %s was ignored",
					k, l.Origin))
				continue
			}
			cur, exists := out.values[k]
			if exists && rank[cur.Source] >= rank[l.Source] {
				continue
			}
			out.values[k] = Value{
				Key: k, Value: v, Source: l.Source, Origin: l.Origin, Locked: locked[k],
			}
		}
	}
	sort.Strings(out.Warnings)
	return out
}

// ---------- roots ----------

// Roots are the four directories, each with its own lifetime.
type Roots struct {
	Config string
	Data   string
	State  string
	Cache  string
}

// DiscoverRoots resolves the four roots.
//
// XDG by default because config, data, state and cache have different
// lifetimes: config belongs in a dotfiles repository, a 500 MB session log does
// not, and cache should be deletable without reconfiguring anything.
//
// DCODE_HOME collapses all four for anyone who prefers one directory, so the
// separation is the default rather than an imposition.
func DiscoverRoots(env func(string) string) (Roots, error) {
	if home := env("DCODE_HOME"); home != "" {
		if !filepath.IsAbs(home) {
			return Roots{}, fmt.Errorf("config: DCODE_HOME must be absolute, got %q", home)
		}
		return Roots{
			Config: home,
			Data:   home,
			State:  filepath.Join(home, "state"),
			Cache:  filepath.Join(home, "cache"),
		}, nil
	}

	home := env("HOME")
	if home == "" {
		return Roots{}, fmt.Errorf("config: HOME is not set and DCODE_HOME was not given")
	}

	pick := func(explicit, xdg, unixDefault, darwinDefault string) string {
		if v := env(explicit); v != "" {
			return v
		}
		if v := env(xdg); v != "" {
			return filepath.Join(v, "dcode")
		}
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, darwinDefault)
		}
		return filepath.Join(home, unixDefault)
	}

	const appSupport = "Library/Application Support/dcode"
	return Roots{
		Config: pick("DCODE_CONFIG_DIR", "XDG_CONFIG_HOME", ".config/dcode", appSupport),
		Data:   pick("DCODE_DATA_DIR", "XDG_DATA_HOME", ".local/share/dcode", appSupport),
		State:  pick("DCODE_STATE_DIR", "XDG_STATE_HOME", ".local/state/dcode", appSupport),
		Cache:  pick("DCODE_CACHE_DIR", "XDG_CACHE_HOME", ".cache/dcode", "Library/Caches/dcode"),
	}, nil
}

// Ensure creates a root on demand with owner-only permissions. Config and state
// hold the user's instructions and history.
func Ensure(dir string) error {
	if dir == "" {
		return fmt.Errorf("config: empty directory")
	}
	return os.MkdirAll(dir, 0o700)
}

// ---------- credential refusal ----------

var credentialKey = regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|credential)`)

// CheckNoCredentials rejects a config file that carries something shaped like a
// secret.
//
// The check runs over every key including unknown sections, because an unknown
// section is exactly where a secret would slip through unnoticed. Config is
// meant to be versioned and synced, which is what makes this the most common
// leak there is.
func CheckNoCredentials(values map[string]string, origin string) error {
	var found []string
	for k := range values {
		if credentialKey.MatchString(k) {
			found = append(found, k)
		}
	}
	if len(found) == 0 {
		return nil
	}
	sort.Strings(found)
	return fmt.Errorf(
		"config: %s contains what looks like a credential (%s). "+
			"Credentials come from the environment, never from a config file — "+
			"this file is meant to be versioned and synced. Set DCODE_API_KEY instead",
		origin, strings.Join(found, ", "))
}

// ---------- instruction discovery ----------

// InstructionFile is one discovered instruction file.
type InstructionFile struct {
	Path   string
	Source string
	Text   string
}

// Instruction file names, in ascending precedence within a directory.
//
// AGENTS.md is the emerging cross-tool convention, so rules a user already
// wrote for another agent are picked up. DCODE.md carries anything meant for
// this tool alone and therefore wins.
var InstructionNames = []string{"AGENTS.md", "DCODE.md"}

// DiscoverInstructions walks from the workspace root down to dir, collecting
// instruction files.
//
// It never reads above the workspace root: in a nested monorepo that would
// silently pull in a neighbouring project's rules and make behaviour
// inexplicable.
func DiscoverInstructions(workspace, dir string, names []string, maxBytes, maxDepth int) ([]InstructionFile, error) {
	if len(names) == 0 {
		names = InstructionNames
	}
	ws, err := filepath.Abs(workspace)
	if err != nil {
		return nil, err
	}
	target, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	rel, err := filepath.Rel(ws, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		// Outside the workspace: only the root itself is considered.
		rel = "."
	}

	chain := []string{ws}
	if rel != "." {
		cur := ws
		for i, part := range strings.Split(rel, string(filepath.Separator)) {
			if maxDepth > 0 && i >= maxDepth {
				break
			}
			cur = filepath.Join(cur, part)
			chain = append(chain, cur)
		}
	}

	var out []InstructionFile
	for i, d := range chain {
		source := "directory"
		if i == 0 {
			source = "project"
		}
		for _, name := range names {
			p := filepath.Join(d, name)
			raw, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			text := string(raw)
			if maxBytes > 0 && len(raw) > maxBytes {
				// Truncating in silence is worse than not reading at all: the
				// user would believe a rule is in force when it is not.
				text = string(raw[:maxBytes]) + fmt.Sprintf(
					"\n\n<!-- truncated at %d bytes; %d more were not loaded -->",
					maxBytes, len(raw)-maxBytes)
			}
			out = append(out, InstructionFile{Path: p, Source: source, Text: text})
		}
	}
	return out, nil
}
