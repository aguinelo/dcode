package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolver turns a path as the model wrote it into an Access the boundary can
// be checked against. This is the only part of the package that touches the
// filesystem, kept apart so Evaluate stays pure.
type Resolver struct {
	// Workspace is the trust root, already canonical.
	Workspace string
}

// NewResolver canonicalises the workspace root once. Everything downstream
// compares against the result, so a workspace reached through a symlink does
// not silently look like an escape.
func NewResolver(workspace string) (*Resolver, error) {
	if workspace == "" {
		return nil, fmt.Errorf("policy: workspace is required")
	}
	if !filepath.IsAbs(workspace) {
		return nil, fmt.Errorf("policy: workspace %q must be absolute", workspace)
	}
	real, err := canonical(workspace)
	if err != nil {
		return nil, fmt.Errorf("policy: workspace %q: %w", workspace, err)
	}
	return &Resolver{Workspace: real}, nil
}

// Resolve turns path into an Access.
//
// Relative paths resolve against the workspace, never the process working
// directory: the daemon's cwd is an implementation detail the model has no
// reason to know about, and letting it matter would make behaviour depend on
// how the server was launched.
//
// Symlinks are followed to their target before the boundary is checked. Doing
// it the other way round means the boundary falls to a single `ln -s`.
func (r *Resolver) Resolve(path string, write bool) (Access, error) {
	if path == "" {
		return Access{}, fmt.Errorf("policy: empty path")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.Workspace, path)
	}
	real, err := canonical(path)
	if err != nil {
		return Access{}, err
	}
	return Access{Path: real, Write: write}, nil
}

// InWorkspace reports containment by path component, never by string prefix.
//
// A prefix comparison is the most common boundary bug there is: it lets
// /home/user/proj2 pass for /home/user/proj.
func (r *Resolver) InWorkspace(a Access) bool {
	rel, err := filepath.Rel(r.Workspace, a.Path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	// Rel yields a path starting with ".." exactly when the target sits
	// outside, which is the containment answer we want.
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

// canonical resolves symlinks, walking up to the nearest existing ancestor when
// the path itself does not exist yet. Creating a new file is legitimate work,
// so a missing leaf must not be an error — but its parent still has to be
// inside the boundary.
func canonical(path string) (string, error) {
	path = filepath.Clean(path)
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real, nil
	}

	dir, leaf := filepath.Split(path)
	dir = filepath.Clean(dir)
	var trailing []string
	for {
		if real, err := filepath.EvalSymlinks(dir); err == nil {
			parts := append([]string{real, leaf}, trailing...)
			return filepath.Join(parts...), nil
		}
		if _, err := os.Lstat(dir); err == nil {
			// It exists but cannot be evaluated — a broken symlink loop. Not
			// something to guess about.
			return "", fmt.Errorf("policy: cannot resolve %q", path)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root without finding anything real. Clean is the
			// best available answer and stays absolute.
			return path, nil
		}
		trailing = append([]string{leaf}, trailing...)
		dir, leaf = parent, filepath.Base(dir)
	}
}
