package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/loop/qualifier"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/sandbox"
	"github.com/aguinelo/dcode/internal/session"
)

// MaxProposals is how many times one qualifying turn may propose.
//
// Bounded because every proposal is kept and every one reaches the operator:
// a turn that kept re-proposing would grow the history without end, and a
// review nobody can read to the bottom defends nothing. Eight is far above
// what a turn fixing a broken command or narrowing a loose one needs.
const MaxProposals = 8

// ErrTooManyProposals is a qualifying turn that tried to propose past
// MaxProposals. It is refused to the model with the reason and it fails the
// commit that follows: dropping the oldest to make room would take away the
// very sequence the history exists to show.
var ErrTooManyProposals = errors.New("too many proposals")

// Proposal is a definition of done the model derived, recorded and not yet
// written down.
//
// Recorded rather than written because the turn that produces it runs in plan
// mode: working out what you will be measured by is reading, and read-only
// denies every write with no exception. The loop takes it from here — it
// measures the criteria under the boundary the WORK will run under, which is
// also the only place they can actually run, and writes the file.
type Proposal struct {
	Spec      string
	Criteria  []qualifier.Proposed
	Protected []string
	// Sequence is its place in the turn, from 1, and At when it arrived. Both
	// are set when it is recorded, never by whoever proposed it.
	Sequence int
	At       time.Time
}

// Proposals is where a qualifying session keeps what the model proposed.
//
// Every proposal, in order. The last one is what gets measured and written;
// the earlier ones go to the operator with it, because narrowing a criterion
// after seeing it pass is the right move, swapping it for a trivial one is the
// wrong move, and only the sequence tells the two apart.
type Proposals struct {
	mu   sync.Mutex
	now  func() time.Time
	kept []*Proposal
	// over is set by a proposal refused at the ceiling, and it outlives Take
	// only as the error Take returns.
	over bool
}

// Take removes every recorded proposal and returns them, oldest first. Nil
// when there is none.
//
// Take rather than Get: committing is the end of their life, and a proposal
// that survived being written would be written again by the next commit. A
// turn that crossed MaxProposals comes back as ErrTooManyProposals, with the
// history still returned so the caller can say what it held.
func (p *Proposals) Take() ([]*Proposal, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out, over := p.kept, p.over
	p.kept, p.over = nil, false
	if over {
		return out, fmt.Errorf("the qualifying turn proposed more than %d times, and nothing was written: "+
			"read the earlier proposals in its log and qualify the folder again: %w", MaxProposals, ErrTooManyProposals)
	}
	return out, nil
}

// put records the next proposal, or refuses it at the ceiling.
func (p *Proposals) put(v *Proposal) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.kept) >= MaxProposals {
		p.over = true
		return fmt.Errorf("this turn has already proposed %d times, the most one qualifying turn may; "+
			"this proposal was not recorded and the turn will fail: %w", MaxProposals, ErrTooManyProposals)
	}
	now := p.now
	if now == nil {
		now = time.Now
	}
	v.Sequence, v.At = len(p.kept)+1, now()
	p.kept = append(p.kept, v)
	return nil
}

// earlierOf is the proposals before the last, as the wire carries them: as the
// model sent them, with nothing measured, because only the last was run.
func earlierOf(every []*Proposal) []protocol.EarlierProposal {
	if len(every) < 2 {
		return nil
	}
	out := make([]protocol.EarlierProposal, 0, len(every)-1)
	for _, p := range every[:len(every)-1] {
		e := protocol.EarlierProposal{
			Sequence:   p.Sequence,
			ProposedAt: p.At,
			Criteria:   make([]protocol.ProposedCriterion, 0, len(p.Criteria)),
			Protected:  p.Protected,
		}
		for _, c := range p.Criteria {
			e.Criteria = append(e.Criteria, protocol.ProposedCriterion{
				Name: c.Name, Command: c.Command, ExitCode: c.ExitCode,
				Expects: string(c.Expects), Why: c.Why,
			})
		}
		out = append(out, e)
	}
	return out
}

// earlierSummary is the same history in the prose a terminal reads, below the
// summary of what was written. Empty when the turn proposed once.
func earlierSummary(earlier []protocol.EarlierProposal) string {
	if len(earlier) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n\nThe turn proposed %d times; the last is what was written. Earlier, oldest first:",
		len(earlier)+1)
	for _, e := range earlier {
		fmt.Fprintf(&b, "\n\n  #%d at %s", e.Sequence, e.ProposedAt.Format(time.TimeOnly))
		for _, c := range e.Criteria {
			fmt.Fprintf(&b, "\n    %-24s %s", c.Name, c.Command)
			if c.Expects != "" {
				b.WriteString(" (expected to " + c.Expects + ")")
			}
		}
	}
	return b.String()
}

// holdProposals keeps a qualifying session's proposals until the loop takes them.
func (d *Daemon) holdProposals(id string, p *Proposals) {
	d.proposalMu.Lock()
	if d.proposalBy == nil {
		d.proposalBy = map[string]*Proposals{}
	}
	d.proposalBy[id] = p
	d.proposalMu.Unlock()
}

func (d *Daemon) proposals(id string) *Proposals {
	d.proposalMu.Lock()
	defer d.proposalMu.Unlock()
	return d.proposalBy[id]
}

// commitDone writes the last of what a qualifying session proposed, and
// announces it with the earlier ones.
//
// The measurement happens HERE and not inside that session, under the boundary
// the work will actually run under. A qualifying turn is read-only, and
// measuring there would call a criterion broken because the sandbox refused it
// a cache directory — a proposal born with a false measurement is worse than
// none.
func (d *Daemon) commitDone(ctx context.Context, sessionID string) (protocol.CommitDoneResponse, error) {
	sess, err := d.manager.Get(sessionID)
	if err != nil {
		return protocol.CommitDoneResponse{}, err
	}
	holder := d.proposals(sessionID)
	if holder == nil {
		return protocol.CommitDoneResponse{}, protocol.Errorf(protocol.CodeInvalidInput,
			"session %s is not a qualifying session", sessionID)
	}
	every, terr := holder.Take()
	if terr != nil {
		// The ceiling, said rather than absorbed: writing the last proposal of
		// a turn that kept re-proposing would hand the person a ruler with its
		// history cut off.
		return protocol.CommitDoneResponse{}, protocol.Errorf(protocol.CodeInvalidInput, "%v", terr)
	}
	if len(every) == 0 {
		return protocol.CommitDoneResponse{}, protocol.Errorf(protocol.CodeInvalidInput,
			"nothing was proposed for %s", sessionID)
	}

	opts, oerr := d.optionsFor(sess.Workspace)
	if oerr != nil {
		return protocol.CommitDoneResponse{}, oerr
	}
	sb, serr := qualifyingSandbox(opts)
	if serr != nil {
		return protocol.CommitDoneResponse{}, serr
	}
	p := every[len(every)-1]
	done, cerr := CommitProposal(ctx, p, criterionRunner(sb, opts), opts.DoneTimeout)
	if cerr != nil {
		return protocol.CommitDoneResponse{}, cerr
	}
	// Announced only now, with the file on disk: the notice says there is a
	// file to review, and it waits for nothing — the turn that proposed has
	// already ended.
	// With every earlier proposal of the turn: the history is the defence
	// against a re-proposal that swapped a criterion for a trivial one.
	done.Proposal.ProposalID = session.NewID(time.Now, randomUint32)
	done.Proposal.Sequence, done.Proposal.ProposedAt = p.Sequence, p.At
	done.Proposal.Earlier = earlierOf(every)
	sess.Emit(protocol.EventDoneProposed, done.Proposal)
	return protocol.CommitDoneResponse{
		Path:      done.Proposal.Path,
		Summary:   done.Summary + earlierSummary(done.Proposal.Earlier),
		Criteria:  len(p.Criteria),
		Proposals: len(every),
	}, nil
}

// qualifyingSandbox builds the boundary a proposal is measured under: the
// project's own, not the read-only one the proposing turn ran in.
func qualifyingSandbox(opts Options) (sandbox.Sandbox, error) {
	return sandbox.New(sandboxConfig(opts, func() bool { return opts.AllowNetwork }), opts.SandboxMode)
}
