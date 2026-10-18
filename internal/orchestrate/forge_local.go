package orchestrate

// The local forge: no origin, or orchestrator.forge = local. The squash is
// made in the integration place and main moves by a conditional update-ref —
// the lock on the measured commit, without a server. What it loses is the PR
// to read, the forge's CI and the server's lock; forge.local says so.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.6.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// localLoses is what forge.local says the first time a project falls to it.
const localLoses = "this project merges locally, with no forge: there is no pull request to read, no CI on " +
	"the forge, and no lock on the measured commit held by a server — the lock is a conditional update of " +
	"main here, and main's health is the project's gate run on the merged commit"

type localForge struct {
	root, main string
	in         *Integration
	gate       Gate
}

func newLocal(s ForgeSetup) (Forge, error) {
	if s.Integration == nil {
		return nil, errors.New("orchestrate: the local forge squashes in the integration place, and none was given")
	}
	return &localForge{root: s.Root, main: s.Main, in: s.Integration, gate: s.Gate}, nil
}

func (l *localForge) Name() string { return ForgeLocal }

func (l *localForge) Tips(ctx context.Context, branch string) (string, string, error) {
	main, err := forgeGit(ctx, l.root, "rev-parse", "--verify", "refs/heads/"+l.main)
	if err != nil {
		return "", "", err
	}
	head, err := forgeGit(ctx, l.root, "rev-parse", "--verify", "refs/heads/"+branch)
	if err != nil {
		return "", "", err
	}
	return main, head, nil
}

// OpenPR opens nothing and pushes nothing: there is nowhere to.
func (l *localForge) OpenPR(context.Context, string, string, string) (protocol.ChildPR, error) {
	return protocol.ChildPR{}, nil
}

// Merge squashes measured onto the main it was measured over, and moves main
// only if main is still there. Main checked out in any worktree is refused,
// saying where: moving a branch under a checkout leaves that tree lying.
func (l *localForge) Merge(ctx context.Context, c protocol.Child, measured string) (string, error) {
	if c.Gate == nil || !exactCommit.MatchString(c.Gate.Base) || !exactCommit.MatchString(measured) {
		return "", fmt.Errorf("orchestrate: %s has no measured commit over an exact main to merge", c.ID)
	}
	where, err := l.checkedOut(ctx)
	if err != nil {
		return "", err
	}
	if where != "" {
		return "", &forgeRefusal{reason: fmt.Sprintf("%s is checked out in %s; a local merge moves %s, and "+
			"would leave that tree behind it. Switch it to another branch, and the merge can run",
			l.main, where, l.main)}
	}
	old := c.Gate.Base
	s, err := l.in.squash(ctx, old, measured, c.Title+"\n\nMeasured: "+measured)
	if err != nil {
		return "", err
	}
	if _, err := forgeGit(ctx, l.root, "update-ref", "refs/heads/"+l.main, s, old); err != nil {
		return "", &forgeRefusal{reason: fmt.Sprintf("%s is no longer at %s, the main the gate measured over; "+
			"it was not moved (%v)", l.main, short(old), err)}
	}
	return s, nil
}

// checkedOut is the worktree that has main checked out, or "".
func (l *localForge) checkedOut(ctx context.Context) (string, error) {
	out, err := forgeGit(ctx, l.root, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	var tree string
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			tree = p
		}
		if line == "branch refs/heads/"+l.main {
			return tree, nil
		}
	}
	return "", nil
}

// MainHealth is the project's gate run on commit: with no CI on a forge, the
// gate is what says whether main passes. No gate declared is unknown, which
// stops the queue: nothing says main is green.
func (l *localForge) MainHealth(ctx context.Context, commit string) (protocol.MainHealth, error) {
	h := protocol.MainHealth{Commit: commit, Health: healthUnknown, Checked: time.Now().UTC()}
	r, err := l.gate.Measure(ctx, l.root, commit)
	h.Checked = time.Now().UTC()
	switch {
	case r.Result == GateUndeclared:
		return h, nil
	case err != nil:
		return h, err
	case r.Result == GateGreen:
		h.Health = healthGreen
	default:
		h.Health = healthRed
	}
	return h, nil
}
