package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/protocol"
)

// `/loop` in the terminal: the daemon conducts it, this client follows it.
//
// The sequence — survey, qualify, stop for a review, work each folder, the
// verdict of the whole run — is loopcommand.Runner's, in the daemon, so every
// client sees one loop. What lives here is how this client says it: the plan,
// the line for each session the run opens, each proposal written, and the end.
// The text is the text this client always said; only who decides moved.

// loopFollow is the daemon's run this client is following.
type loopFollow struct {
	id string
	// planned, warned and announced keep each thing said once, however many
	// frames repeat it: a frame is the whole run as it is now.
	planned   bool
	warned    int
	announced map[string]bool
	reviewed  bool
	// session is the run's session on screen, so interrupting it stops the
	// run and interrupting anything else does not.
	session string
	// interrupted is this client having stopped the run because the person
	// interrupted its turn, so the end can say which stop it was.
	interrupted bool
	// awaiting is the run waiting for a review: its session is the person's
	// to use, so text typed there goes.
	awaiting bool
	msgs     chan tea.Msg
	ctx      context.Context
	cancel   context.CancelFunc
}

// loopNoteMsg is something the follower has to say. It keeps the follow going
// unless ended says the follower has stopped.
type loopNoteMsg struct {
	text  string
	ended bool
}

// loopStartedMsg is a run this client started, or continued after a review.
type loopStartedMsg struct {
	run       protocol.LoopRun
	continued bool
}

// loopChangedMsg is the run as it is now, with the session it moved into when
// it moved into one.
type loopChangedMsg struct {
	run     protocol.LoopRun
	session *protocol.Session
}

// sameLoopArgs compares what was typed, word for word, ignoring spacing.
func sameLoopArgs(a, b string) bool {
	return strings.Join(strings.Fields(a), " ") == strings.Join(strings.Fields(b), " ")
}

// startLoop asks the daemon to run `/loop`, or to go on with the loop waiting
// for this review.
//
// "Review it, then /loop again" is what this client tells a person after a
// proposal, and typing it again continues the loop waiting in this workspace —
// when it is the same loop. A different one is refused, saying which is
// waiting, rather than silently continuing a run the person did not mean.
func (p *program) startLoop(args string) tea.Cmd {
	ws, model, sandbox, lang := p.model.Workspace, p.opts.Model, p.model.Sandbox, p.model.Lang
	return func() tea.Msg {
		runs, err := p.opts.Transport.ListLoops(p.ctx, ws)
		if err != nil {
			return noteMsg("could not ask the daemon about its loops: " + err.Error())
		}
		for _, run := range runs {
			if !run.Live() {
				continue
			}
			if run.State != protocol.LoopAwaitingReview {
				// The same loop, still working — typed again after switching
				// away: follow it again rather than ask for a second one.
				if sameLoopArgs(run.Args, args) {
					return loopStartedMsg{run: run, continued: true}
				}
				continue
			}
			if !sameLoopArgs(run.Args, args) {
				return noteMsg(fmt.Sprintf(Text(lang).CmdLoopWaiting, run.Args))
			}
			got, err := p.opts.Transport.ContinueLoop(p.ctx, run.ID)
			if err != nil {
				return noteMsg("could not continue the loop: " + err.Error())
			}
			return loopStartedMsg{run: got, continued: true}
		}
		run, err := p.opts.Transport.StartLoop(p.ctx, protocol.StartLoopRequest{
			Workspace: ws, Args: args, Model: model, SandboxMode: sandbox,
		})
		if err != nil {
			return noteMsg("could not start the loop: " + err.Error())
		}
		return loopStartedMsg{run: run}
	}
}

// onLoopStarted follows a run: one stream, read in its own goroutine, with the
// session the run moves into looked up there so the update loop never waits.
//
// Continuing answers with the run as it was when it went on, and the follow on
// screen keeps hearing the run until that answer arrives. A run that ended in
// between has had its end said by that follow; following it again would say
// the end twice.
func (p *program) onLoopStarted(msg loopStartedMsg) tea.Cmd {
	if msg.continued && msg.run.ID == p.loopOver {
		return nil
	}
	p.stopFollowingLoop()
	f := &loopFollow{id: msg.run.ID, announced: map[string]bool{}, msgs: make(chan tea.Msg, 64)}
	if msg.continued {
		// Already said when the run stopped for the review.
		f.planned, f.reviewed, f.warned = true, true, len(msg.run.Warnings)
		for _, s := range msg.run.Specs {
			if s.Proposal != "" {
				f.announced[s.Path] = true
			}
		}
	}
	ctx, cancel := context.WithCancel(p.ctx)
	f.ctx, f.cancel = ctx, cancel
	p.loop = f
	go p.followLoop(ctx, f)
	return p.onLoopChanged(loopChangedMsg{run: msg.run})
}

func (p *program) stopFollowingLoop() {
	if p.loop != nil {
		p.loop.cancel()
		p.loop = nil
	}
}

// followLoop turns the stream of runs into messages about this one.
func (p *program) followLoop(ctx context.Context, f *loopFollow) {
	changes, errs := p.opts.Transport.WatchLoops(ctx)
	send := func(m tea.Msg) bool {
		select {
		case f.msgs <- m:
			return true
		case <-ctx.Done():
			return false
		}
	}
	lastSession := ""
	for {
		select {
		case ch, open := <-changes:
			if !open {
				// The stream closes errs before changes; whichever the select
				// saw first, the reason is said and the follow ends.
				text := "the loop could not be followed: the stream ended"
				select {
				case err, ok := <-errs:
					if ok && err != nil {
						text = "the loop could not be followed: " + err.Error()
					}
				default:
				}
				send(loopNoteMsg{text: text, ended: true})
				return
			}
			run := loopIn(ch, f.id)
			if run == nil {
				continue
			}
			m := loopChangedMsg{run: *run}
			if run.SessionID != "" && run.SessionID != lastSession {
				s, err := p.opts.Transport.GetSession(ctx, run.SessionID)
				if err != nil {
					if !send(loopNoteMsg{text: "could not open the loop's session: " + err.Error()}) {
						return
					}
				} else {
					m.session, lastSession = &s, run.SessionID
				}
			}
			if !send(m) || !run.Live() {
				return
			}
		case err, open := <-errs:
			text := "the loop could not be followed: the stream ended"
			if open && err != nil {
				text = "the loop could not be followed: " + err.Error()
			}
			send(loopNoteMsg{text: text, ended: true})
			return
		case <-ctx.Done():
			return
		}
	}
}

func loopIn(ch protocol.LoopChange, id string) *protocol.LoopRun {
	if ch.Loop != nil && ch.Loop.ID == id {
		return ch.Loop
	}
	for i := range ch.Loops {
		if ch.Loops[i].ID == id {
			return &ch.Loops[i]
		}
	}
	return nil
}

// waitForLoop delivers the next message about the run being followed.
func (p *program) waitForLoop() tea.Cmd {
	f := p.loop
	if f == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case m := <-f.msgs:
			return m
		case <-f.ctx.Done():
			// The follow was replaced or stopped: nothing more comes, and
			// waiting on would leave this goroutine behind for good.
			return nil
		}
	}
}

// onLoopNote says what the follower had to say, and keeps following unless
// the follower has stopped.
func (p *program) onLoopNote(msg loopNoteMsg) tea.Cmd {
	p.loopNote(msg.text)
	if msg.ended {
		p.stopFollowingLoop()
		return nil
	}
	return p.waitForLoop()
}

// onLoopChanged says what moved, once each: the plan, the session the run
// moved into, each proposal written, and the end.
func (p *program) onLoopChanged(msg loopChangedMsg) tea.Cmd {
	f := p.loop
	if f == nil || msg.run.ID != f.id {
		return nil
	}
	run := msg.run
	var cmds []tea.Cmd

	// The plan, before any of it runs: every folder, not only the pending ones.
	// A goal that found no folder is qualified at the workspace's anchor, and
	// has no plan to draw.
	if run.Goal != "" && !f.planned && len(run.Specs) > 0 &&
		!(len(run.Specs) == 1 && run.Specs[0].Path == protocol.WorkspaceDoneDir) {
		f.planned = true
		p.loopNote(LoopPlan(loopFolders(run.Specs), Text(p.opts.Lang)))
	}

	if msg.session != nil && msg.session.ID != p.opts.SessionID {
		cmds = append(cmds, p.loopOpened(run, *msg.session))
	}

	for _, w := range run.Warnings[min(f.warned, len(run.Warnings)):] {
		p.loopNote("loop: " + w)
	}
	f.warned = len(run.Warnings)

	proposed := 0
	for _, s := range run.Specs {
		if s.Proposal == "" {
			continue
		}
		if s.Path != protocol.WorkspaceDoneDir {
			proposed++
		}
		if f.announced[s.Path] {
			continue
		}
		f.announced[s.Path] = true
		// The command that goes on is the one that started the run, as typed:
		// for a goal or a bare word that is the sentence, not the folder.
		p.loopNote(fmt.Sprintf(Text(p.opts.Lang).CmdLoopProposed, s.ProposalCriteria, s.Proposal, run.Args) +
			"\n\n" + s.ProposalSummary)
	}
	// A goal's pass over the backlog ends with one sentence for the review.
	if run.State == protocol.LoopAwaitingReview && run.Goal != "" && proposed > 0 && !f.reviewed {
		f.reviewed = true
		p.loopNote(fmt.Sprintf(Text(p.opts.Lang).CmdLoopReviewAll, proposed))
	}

	f.awaiting = run.State == protocol.LoopAwaitingReview
	if !run.Live() {
		p.loopEnded(run)
		p.loopOver = run.ID
		p.stopFollowingLoop()
		// What the person queued while the loop ran waited for it. It goes
		// on its own only when the loop finished: after a stop or a failure
		// nothing fires by itself, and the person says when (owner's
		// decision).
		if len(p.model.Queue) > 0 {
			if run.State == protocol.LoopFinished && p.model.State == protocol.SessionStateIdle {
				m, text := p.model.DrainQueue()
				p.model = m
				cmds = append(cmds, p.submit(text))
			} else if run.State != protocol.LoopFinished {
				p.loopNote(fmt.Sprintf(Text(p.opts.Lang).CmdLoopQueueKept, len(p.model.Queue)))
			}
		}
		return tea.Batch(cmds...)
	}
	cmds = append(cmds, p.waitForLoop())
	return tea.Batch(cmds...)
}

// loopOpened is the switch to the session the run moved into, plus the one
// line that says what was loaded. The note goes in AFTER the view is rebuilt,
// or it would be written into the view the switch discards. The turn is the
// daemon's to submit.
func (p *program) loopOpened(run protocol.LoopRun, s protocol.Session) tea.Cmd {
	t := Text(p.opts.Lang)
	spec, qualify := "", false
	for _, sp := range run.Specs {
		if sp.SessionID == s.ID {
			spec = sp.Path
			qualify = sp.Phase == protocol.SpecQualifying || sp.Phase == protocol.SpecProposed
		}
	}
	queued := p.model.Queue
	p.opts.SessionID = s.ID
	p.model = NewModel(s.ID, s.Workspace, s.Model, s.SandboxMode, p.opts.Lang)
	p.model.Sessions = p.opts.Sessions
	// The queue goes with the person, not with the session it was typed in:
	// sent to the previous folder it would start a second writer in the tree
	// and a reply nobody sees, and dropped it would vanish without a word.
	p.model.Queue = queued
	text := fmt.Sprintf(t.CmdLoopOpened, spec, s.DoneCriteria)
	if qualify {
		text = fmt.Sprintf(t.CmdLoopQualifying, spec)
	} else if s.DoneCriteria == 0 {
		// Zero is the answer that needs the sentence. A session with no
		// definition of done reports done at the end of the first turn, and
		// someone who typed /loop expecting one has to be told now.
		text = fmt.Sprintf(t.CmdLoopEmpty, spec)
	}
	p.loopNote(text)
	if len(queued) > 0 {
		p.loopNote(fmt.Sprintf(t.CmdLoopQueueHeld, len(queued), spec))
	}
	if p.loop != nil {
		p.loop.session = s.ID
	}
	p.attach(s.ID)
	return p.waitForEvent()
}

// loopEnded says how the run ended.
//
// A goal's run says where done stands for the whole run, naming what is left
// and where; a run that worked nothing, or a single folder, says nothing more
// than its session already did — as this client always behaved.
func (p *program) loopEnded(run protocol.LoopRun) {
	t := Text(p.model.Lang)
	switch run.State {
	case protocol.LoopFailed:
		p.loopNote(run.Error)
		return
	case protocol.LoopStopped:
		text := t.CmdLoopStoppedElsewhere
		if p.loop != nil && p.loop.interrupted {
			text = t.CmdLoopStopped
		}
		if run.Error != "" {
			text += "\n" + run.Error
		}
		p.loopNote(text)
		return
	}
	v := run.Verdict
	if run.Goal == "" || v == nil || v.Worked == 0 {
		return
	}
	text := fmt.Sprintf(t.CmdLoopFinished, v.Worked, plural(v.Worked, t.LoopSpecOne, t.LoopSpecMany))
	if v.Total > 0 {
		text += " · " + fmt.Sprintf(t.CmdLoopStanding, v.Met, v.Total)
	}
	// What is left, by name: a count says how much; the names say what to do
	// next, and which specs they belong to says where.
	if len(v.Unmet) > 0 {
		text += "\n" + t.CompletionUnmet + " " + strings.Join(v.Unmet, ", ")
	}
	if len(v.Unchecked) > 0 {
		text += "\n" + t.CompletionUnchecked + " " + strings.Join(v.Unchecked, ", ")
	}
	if len(v.Unfinished) > 0 {
		text += "\n" + t.CmdLoopUnfinished + " " + strings.Join(v.Unfinished, ", ")
	}
	p.loopNote(text)
}

func (p *program) loopNote(text string) {
	p.model.Entries = append(p.model.Entries, Entry{Kind: KindNote, Summary: text, Expanded: true})
}

// loopFolders is the plan as the survey left it: pending is what the folder's
// own criteria said, never a phase reached later.
func loopFolders(specs []protocol.LoopSpecRun) []protocol.SpecFolder {
	out := make([]protocol.SpecFolder, 0, len(specs))
	for _, s := range specs {
		out = append(out, protocol.SpecFolder{
			Path: s.Path, Criteria: s.Criteria, Unmet: s.Unmet, Unavailable: s.Unavailable, Error: s.Error,
			Pending: s.Error == "" && (s.Criteria == 0 || s.Unmet > 0 || s.Unavailable > 0),
		})
	}
	return out
}

// interruptLoop stops the person's own loop when they interrupt its turn: the
// loop ends stopped, rather than failing on a turn that ended interrupted. It
// used to skip to the next folder; the daemon does not, and says so.
func (p *program) interruptLoop() (tea.Cmd, bool) {
	f := p.loop
	if f == nil || f.session != p.opts.SessionID {
		return nil, false
	}
	id := f.id
	f.interrupted = true
	return func() tea.Msg {
		if _, err := p.opts.Transport.StopLoop(p.ctx, id, protocol.StopNow); err != nil {
			return noteMsg("could not stop the loop: " + err.Error())
		}
		return nil
	}, true
}

// loopHolds reports a live loop in the session on screen, between turns of
// the daemon's: text sent there now would start a turn in the folder it is
// leaving, so it is held. A loop waiting for a review holds nothing.
func (p *program) loopHolds() bool {
	f := p.loop
	return f != nil && f.session == p.opts.SessionID && !f.awaiting
}
