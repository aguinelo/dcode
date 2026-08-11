package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
)

// GrantsFile is where standing permissions live, under the USER's config root.
//
// Never in the workspace, and that placement is the security property rather
// than a filing preference: a grants file inside a project would let a
// repository arrive pre-approved. Cloning something would then grant it the
// network before anyone read a line of it, which is the opposite of asking.
const GrantsFile = "grants.toml"

// Grants is what the user has already permitted, so they are not asked again.
//
// Without it the product has only two states, and both are wrong: refuse the
// network always, or ask about it once per command. The second is worse than it
// sounds — a shell command is opaque, so the question is true of nearly every
// one of them, and a question that always fires is one people learn to answer
// without reading.
type Grants struct {
	// networkAlways is the answer that covers every project, including ones
	// that do not exist yet.
	networkAlways bool
	// networkProjects are the workspaces answered one at a time, by resolved
	// path: two spellings of one directory are one project, or the user is
	// asked again about a place they already decided on.
	networkProjects map[string]struct{}
}

// Network reports whether this workspace may reach the network without asking.
func (g Grants) Network(workspace string) bool {
	if g.networkAlways {
		return true
	}
	_, ok := g.networkProjects[canonicalWorkspace(workspace)]
	return ok
}

// GrantNetwork remembers the answer for one workspace.
func (g Grants) GrantNetwork(workspace string) Grants {
	out := g.clone()
	out.networkProjects[canonicalWorkspace(workspace)] = struct{}{}
	return out
}

// GrantNetworkAlways remembers the answer for every workspace.
func (g Grants) GrantNetworkAlways() Grants {
	out := g.clone()
	out.networkAlways = true
	return out
}

func (g Grants) clone() Grants {
	out := Grants{networkAlways: g.networkAlways, networkProjects: map[string]struct{}{}}
	for k := range g.networkProjects {
		out.networkProjects[k] = struct{}{}
	}
	return out
}

// LoadGrants reads the standing permissions from a config root.
//
// An absent file is the ordinary case on a fresh machine and grants nothing. A
// file that cannot be parsed is an ERROR and still grants nothing: a record
// nobody can interpret must not be read as "everything is permitted", and
// failing closed is the only safe reading of it.
func LoadGrants(root string) (Grants, error) {
	empty := Grants{networkProjects: map[string]struct{}{}}

	data, err := os.ReadFile(filepath.Join(root, GrantsFile))
	if os.IsNotExist(err) {
		return empty, nil
	}
	if err != nil {
		return empty, fmt.Errorf("config: cannot read %s: %w", GrantsFile, err)
	}

	doc, err := ParseSections(string(data), GrantsFile)
	if err != nil {
		return empty, fmt.Errorf("config: %s is unreadable, so nothing is granted: %w", GrantsFile, err)
	}

	out := empty
	net := doc.Values["network"]
	if strings.TrimSpace(net["always"]) == "true" {
		out.networkAlways = true
	}
	for _, p := range policy.SplitList(net["projects"]) {
		if p = strings.TrimSpace(p); p != "" {
			out.networkProjects[canonicalWorkspace(p)] = struct{}{}
		}
	}
	return out, nil
}

// Save writes the standing permissions, owner-only.
//
// The same mode credentials get, for the same reason: this records what the
// user permitted, and a decision anyone on the machine can edit is a decision
// the user did not make.
func (g Grants) Save(root string) error {
	if err := Ensure(root); err != nil {
		return err
	}
	projects := make([]string, 0, len(g.networkProjects))
	for p := range g.networkProjects {
		projects = append(projects, p)
	}
	// Sorted so the file does not churn between saves; a record that reorders
	// itself is one nobody can diff to see what changed.
	sort.Strings(projects)

	var b strings.Builder
	b.WriteString("# Written by dcode. What you have already permitted, so you are\n")
	b.WriteString("# not asked again. Delete a line to be asked about it next time.\n\n")
	b.WriteString("[network]\n")
	fmt.Fprintf(&b, "always = %t\n", g.networkAlways)
	b.WriteString("projects = [")
	for i, p := range projects {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q", p)
	}
	b.WriteString("]\n")

	return os.WriteFile(filepath.Join(root, GrantsFile), []byte(b.String()), 0o600)
}

// canonicalWorkspace resolves a workspace to the name a grant is keyed by.
//
// Symlinks are followed where they can be: the same directory reached two ways
// is one project. Where it cannot be resolved — a path that no longer exists —
// the cleaned form is used, because refusing to key an absent path would mean
// forgetting a grant whenever a project is temporarily unmounted.
func canonicalWorkspace(p string) string {
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return filepath.Clean(p)
}
