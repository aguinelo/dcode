package app

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/orchestrate"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// The tool sets of the orchestrator and of its children.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §10.

// conductorTools adds the orchestrator's tools when the session conducts a
// project. A child never has a Conductor, so it never has them.
func conductorTools(toolset []tools.Tool, opts Options) []tools.Tool {
	if opts.Conductor == nil {
		return toolset
	}
	c := opts.Conductor
	return append(toolset, tools.DelegateTool{Conductor: c}, tools.ChildrenTool{Conductor: c},
		tools.SteerChildTool{Conductor: c}, tools.CloseChildTool{Conductor: c})
}

// childTools is a child's tool set: its shell refuses git that writes, and
// every other tool's write outside owns is refused, without a question, with
// outside_owns. The shell's own writes are the porcelain check's, after the
// turn (.p §7.1): a shell command is opaque, so nothing here can prove what
// it would touch.
func childTools(toolset []tools.Tool, resolver *policy.Resolver, owns []string) []tools.Tool {
	out := make([]tools.Tool, 0, len(toolset))
	for _, t := range toolset {
		if b, ok := t.(tools.Bash); ok {
			b.Deny = tools.DenyGitWrite
			out = append(out, b)
			continue
		}
		out = append(out, ownedTool{Tool: t, resolver: resolver, owns: owns})
	}
	return out
}

// ownedTool is a tool whose declared writes must sit inside owns.
type ownedTool struct {
	tools.Tool
	resolver *policy.Resolver
	owns     []string
}

func (t ownedTool) Declare(input json.RawMessage) (policy.Request, error) {
	req, err := t.Tool.Declare(input)
	if err != nil {
		return req, err
	}
	for _, a := range req.Paths {
		if !a.Write {
			continue
		}
		// Outside the worktree, or unresolvable, is owned by no one.
		r, rerr := t.resolver.Resolve(a.Path, true)
		inside := rerr == nil && (r.Rel != "" || filepath.Clean(r.Path) == t.resolver.Workspace)
		if !inside || !orchestrate.Owned(t.owns, r.Rel) {
			return policy.Request{}, &tools.ToolError{Tool: t.Name(), Code: protocol.CodeOutsideOwns,
				Reason: "a child writes only the paths it owns; " + a.Path + " is outside them (" + strings.Join(t.owns, ", ") + "). " +
					"Leave it, or say in your report that it needs changing"}
		}
	}
	return req, nil
}
