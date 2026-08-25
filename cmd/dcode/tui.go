package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	xterm "github.com/charmbracelet/x/term"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/session"
	"github.com/aguinelo/dcode/internal/tui"
	"github.com/aguinelo/dcode/pkg/client"
)

// runTUI opens the terminal client.
//
// It attaches to a running daemon when one answers, and otherwise starts its
// own inside this process. The client speaks the protocol either way — the
// embedded daemon is a deployment detail, not a second code path — so a session
// that outgrows one terminal can be handed to `dcode serve` without changing
// anything the client does.
func runTUI(args []string) error {
	fs := flag.NewFlagSet("dcode tui", flag.ContinueOnError)
	var (
		socket    = fs.String("socket", "", "attach to this daemon instead of starting one")
		workspace = fs.String("workspace", "", "workspace root (default: current directory)")
		attach    = fs.String("session", "", "attach to an existing session by id")
		// Continuing and attaching look alike and are not. Attaching joins a
		// session that is still running; continuing starts a new one carrying
		// the conversation of one that ended.
		cont = fs.Bool("continue", false, "continue the most recent session for this workspace")
		// Picking and continuing are also different, and the split is the one
		// people already know from other harnesses: continue takes the last
		// one, resume asks which. Asking is the safer default for a workspace
		// somebody has used all week, and taking the last is the faster one for
		// the session they just closed.
		pick = fs.Bool("resume", false, "choose which recorded session to continue")
	)
	fs.BoolVar(cont, "c", false, "shorthand for --continue")
	fs.BoolVar(pick, "r", false, "shorthand for --resume")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "dcode tui — open the terminal interface\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	ws, err := resolveWorkspace(*workspace)
	if err != nil {
		return err
	}
	opts, resolved, err := loadOptions(ws)
	if err != nil {
		return err
	}

	roots, err := config.DiscoverRoots(os.Getenv)
	if err != nil {
		return err
	}
	// Frozen at start, like the instruction chain: a command file written
	// mid-session must not change what a slash means halfway through.
	commands, err := config.DiscoverCommands(roots, ws, 64<<10)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	path := *socket
	explicit := path != ""
	if path == "" {
		path = app.DefaultSocketPath(os.Getenv)
	}

	c := client.New(path)
	if err := healthy(ctx, c); err != nil {
		if explicit {
			// An address the user named and that does not answer is an error,
			// not an invitation to start something somewhere else.
			return fmt.Errorf("no daemon answered on %s: %w", path, err)
		}
		embedded, cleanup, err := startEmbedded(ctx, opts, recordDir(resolved), recordBudget(resolved))
		if err != nil {
			return err
		}
		defer cleanup()
		c = embedded
	}

	// An id given as a bare argument continues that one without asking, which
	// is what `dcode -r <id>` reads as and what `dcode sessions` prints ids for.
	carry := fs.Arg(0)
	if *cont && carry == "" {
		if carry, err = latestSession(recordDir(resolved), ws); err != nil {
			return err
		}
	}
	if *pick && carry == "" {
		if carry, err = pickSession(ctx, recordDir(resolved), ws, geometry(), resolved); err != nil {
			return err
		}
	}

	sessionID := *attach
	var sess protocol.Session
	if sessionID == "" {
		sess, err = c.CreateSession(ctx, protocol.CreateSessionRequest{
			Workspace:   ws,
			Model:       opts.Model,
			SandboxMode: string(opts.SandboxMode),
			Resume:      carry,
		})
	} else {
		sess, err = c.GetSession(ctx, sessionID)
	}
	if err != nil {
		return err
	}

	geo := geometry()

	return tui.Run(ctx, tui.Options{
		SessionID: sess.ID,
		Workspace: sess.Workspace,
		Model:     sess.Model,
		Sandbox:   sess.SandboxMode,
		Window:    sess.ContextWindow,
		// Everything already in the log is history: continuing replays the
		// whole of the old conversation into the new session.
		Backlog: sess.LastSeq,
		// From what the session HAS, not from 1. A continued conversation
		// longer than the retention window has had its oldest events dropped
		// already, and asking for them is refused — which the client used to
		// turn into a silent exit at the splash screen.
		From:          sess.FirstSeq,
		Transport:     c,
		Geometry:      geo,
		Commands:      commands,
		Sessions:      recordedSessions(recordDir(resolved), ws),
		AcceptsImages: acceptsImages(opts.Model),
		Lookup:        lookup(resolved),
		// Resolved at the edge, once. The client package renders and never
		// reads the environment, the same way it never builds its own palette.
		Lang:   langOf(resolved),
		Notice: versionNotice,
		Update: updater,
	})
}

// geometry reads the terminal once. Built here rather than in the client for
// the reason the language is: the client renders and never reads the
// environment.
func geometry() tui.Geometry {
	w, h := terminalSize()
	geo := tui.DefaultGeometry(w, h)
	geo.Unicode = supportsUnicode(os.Getenv)
	geo.ActivityVerbs = tui.ActivityVerbsEnabled(os.Getenv)
	// Only what the edge can know: the client never reads the environment, and
	// the theme it starts in is already in the defaults.
	geo.Palette.Enabled = tui.ColorEnabled(os.Getenv) && isTerminal(os.Stdout)
	geo.Palette.Depth = tui.ColorDepth(os.Getenv)
	return geo
}

// langOf resolves the interface language at the edge, once. The client package
// renders and never reads the environment, the same way it never builds its own
// palette.
func langOf(resolved config.Resolved) tui.Lang {
	return tui.Resolve(func(k string) string {
		if k == "DCODE_LANG" {
			if v := resolved.String("ui.lang", ""); v != "" {
				return v
			}
		}
		return os.Getenv(k)
	})
}

// pickSession asks which conversation to continue.
//
// An empty answer is not a failure: somebody who opened the list and changed
// their mind gets a fresh session, which is what Esc means everywhere else
// here.
func pickSession(ctx context.Context, dir, ws string, geo tui.Geometry, resolved config.Resolved) (string, error) {
	found, err := session.Browse(dir, ws)
	if err != nil {
		return "", err
	}
	return tui.Pick(ctx, choicesFrom(found), geo, langOf(resolved))
}

// choicesFrom is the listing as a list somebody picks from.
//
// A session nobody asked anything in is nothing to continue, and it is most of
// what a record directory holds: one is written every time the interface opens.
// Offering them would bury the four real conversations under thirty empty ones.
// recordedSessions is the sidebar's list, read once at start.
//
// The same source and the same filter as `dcode -r`, deliberately: two ways of
// listing the conversations of a workspace would eventually disagree about
// which ones exist. An unreadable record directory is no list rather than a
// failure to start — the sidebar is a convenience, and refusing to open the
// interface over it would be the wrong trade.
func recordedSessions(dir, ws string) []tui.SessionChoice {
	found, err := session.Browse(dir, ws)
	if err != nil {
		return nil
	}
	return choicesFrom(found)
}

func choicesFrom(found []session.Summary) []tui.SessionChoice {
	var out []tui.SessionChoice
	for _, s := range found {
		if s.Turns == 0 {
			continue
		}
		out = append(out, tui.SessionChoice{
			ID: s.ID, Title: s.Title, Name: s.Name, Turns: s.Turns, When: s.Started,
		})
	}
	return out
}

// healthy reports whether a daemon answers, with a short deadline: this runs
// before the first frame, and a slow probe is felt as a slow start.
func healthy(ctx context.Context, c *client.Client) error {
	probe, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	return c.Health(probe)
}

// startEmbedded runs a daemon inside this process on a private socket.
//
// Private rather than the default path: two terminals opened without a shared
// daemon would otherwise race to bind the same socket, and the loser would fail
// to start for a reason the user cannot act on.
func startEmbedded(ctx context.Context, base app.Options, recordDir string, budget session.PruneBudget) (*client.Client, func(), error) {
	dir, err := os.MkdirTemp("", "dcode")
	if err != nil {
		return nil, nil, err
	}
	path := filepath.Join(dir, "d.sock")

	// The embedded daemon records too. It did not, and `dcode` and `dcode tui`
	// are the paths people actually use — so the mechanism existed, with a
	// config key and a state directory, and produced nothing for anybody.
	d := app.NewDaemon(app.DaemonOptions{
		SocketPath:     path,
		EventRetention: 10000,
		RecordDir:      recordDir,
		RecordBudget:   budget,
		Base:           base,
	})
	if err := d.Listen(); err != nil {
		os.RemoveAll(dir)
		return nil, nil, err
	}

	serveCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = d.Serve(serveCtx)
	}()

	cleanup := func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		os.RemoveAll(dir)
	}

	c := client.New(path)
	if err := waitReady(ctx, c); err != nil {
		cleanup()
		return nil, nil, err
	}
	return c, cleanup, nil
}

// waitReady polls until the embedded daemon answers. Listen has already bound
// the socket, so this only covers the gap before Serve accepts.
func waitReady(ctx context.Context, c *client.Client) error {
	deadline := time.Now().Add(3 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		if err := healthy(ctx, c); err == nil {
			return nil
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
	if last == nil {
		last = errors.New("timed out")
	}
	return fmt.Errorf("the embedded daemon did not start: %w", last)
}

func isTerminal(f *os.File) bool { return xterm.IsTerminal(f.Fd()) }

// terminalSize falls back to the documented default rather than to something
// tiny: guessing small hides the plan panel on a terminal that could show it.
func terminalSize() (int, int) {
	w, h, err := xterm.GetSize(os.Stdout.Fd())
	if err != nil || w <= 0 || h <= 0 {
		return 100, 30
	}
	return w, h
}

// supportsUnicode decides between the box-drawing glyphs and the ASCII set.
//
// The locale is the only portable signal available before the first frame.
// Guessing wrong towards ASCII is safe; guessing wrong towards Unicode leaves
// the plan panel full of replacement characters.
func supportsUnicode(env func(string) string) bool {
	if v := env("DCODE_ASCII"); v == "1" || v == "true" {
		return false
	}
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := env(k)
		if v == "" {
			continue
		}
		return containsFold(v, "utf-8") || containsFold(v, "utf8")
	}
	return false
}

func containsFold(s, sub string) bool {
	ls, lsub := []byte(s), []byte(sub)
	lower := func(b byte) byte {
		if b >= 'A' && b <= 'Z' {
			return b + 32
		}
		return b
	}
	for i := 0; i+len(lsub) <= len(ls); i++ {
		ok := true
		for j := range lsub {
			if lower(ls[i+j]) != lower(lsub[j]) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// lookup answers `/config <key>` with the value and its provenance, which is
// the only form of the answer that lets a user act on it.
func lookup(r config.Resolved) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := r.Get(key)
		if !ok {
			return "", false
		}
		out := fmt.Sprintf("%s = %s\n  from: %s (%s)", v.Key, v.Value, v.Source, v.Origin)
		if v.Locked {
			out += "\n  locked by the administrator"
		}
		return out, true
	}
}

// latestSession is the most recent recorded session for this workspace.
//
// Filtered by workspace, because --continue means "what I was doing here". An
// unfiltered latest would hand someone the session from another project, which
// is worse than finding none: they would notice none, and might not notice the
// wrong one until the model answered from it.
func latestSession(dir, ws string) (string, error) {
	found, err := session.Browse(dir, ws)
	if err != nil {
		return "", err
	}
	if len(found) == 0 {
		return "", fmt.Errorf("no recorded session for this workspace to continue")
	}
	// The newest session that was actually used, not simply the newest.
	//
	// A record is opened before the first question, so an interface closed
	// without asking anything — or one that failed to open at all — leaves an
	// empty record behind. Continuing that one carries nothing, and the run
	// that did it leaves another empty record for the next: taken literally,
	// `--continue` destroys its own target every time it is used.
	for _, s := range found {
		if s.Turns > 0 {
			return s.ID, nil
		}
	}
	// Distinguished from having no session at all, because they are different
	// situations and the fix for each is different.
	return "", fmt.Errorf("the recorded sessions for this workspace are ones where nothing was asked — nothing to continue")
}

// acceptsImages asks the family that owns this model whether it reads pictures.
//
// Resolved here, once, so the client can refuse before sending rather than
// letting a provider reject the request thirty seconds later — a failure the
// person cannot connect to what they did.
func acceptsImages(model string) bool {
	f, ok := provider.FamilyFor(model, app.Families())
	if !ok {
		return false
	}
	return f.AcceptsImages()
}
