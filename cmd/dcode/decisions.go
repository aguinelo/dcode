package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/orchestrate"
)

// followEvery is how often --follow looks for new lines.
const followEvery = 250 * time.Millisecond

// runDecisions prints a project's decision log.
//
// It reads the file itself, never the daemon: the log has to be readable when
// what failed was the daemon. It never writes — not even the cut of a torn
// last line, which is the orchestrator's to make when it reopens the log.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §9.2.
func runDecisions(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return decisions(ctx, args, os.Getenv, time.Now, os.Stdout, os.Stderr)
}

type decisionsQuery struct {
	child  string
	kinds  map[string]bool
	since  time.Time
	follow bool
	json   bool
}

func decisions(ctx context.Context, args []string, env func(string) string, now func() time.Time,
	stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("dcode decisions", flag.ContinueOnError)
	fs.SetOutput(stderr)
	project := fs.String("project", "", "the project's folder, any path into its tree (default: here)")
	child := fs.String("child", "", "only the decisions about this child")
	kind := fs.String("kind", "", "only these kinds, comma-separated")
	since := fs.Duration("since", 0, "only the decisions of the last `duration` (90m, 24h)")
	follow := fs.Bool("follow", false, "keep printing decisions as they are written")
	raw := fs.Bool("json", false, "the lines as they are in the log, byte for byte")
	fs.Usage = func() {
		fmt.Fprint(stderr, `usage:
  dcode decisions [--project <path>] [--child <id>] [--kind <k>[,<k>]] [--since <dur>] [--follow] [--json]

Reads the orchestrator's decision log of a project, without the daemon.

`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unknown argument %q; try `dcode decisions --help`", fs.Arg(0))
	}
	if *since < 0 {
		return fmt.Errorf("--since %s is in the future; give how far back, like 24h", *since)
	}
	q := decisionsQuery{child: *child, follow: *follow, json: *raw}
	if *since > 0 {
		q.since = now().Add(-*since)
	}
	if *kind != "" {
		q.kinds = map[string]bool{}
		for _, k := range strings.Split(*kind, ",") {
			if k = strings.TrimSpace(k); k != "" {
				q.kinds[k] = true
			}
		}
	}

	path, err := decisionLogPath(ctx, *project, env)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no decision log at %s: no orchestrator has opened this project", path)
	}
	if err != nil {
		return err
	}
	defer f.Close()
	return printDecisions(ctx, path, orchestrate.NewLogReader(f), q, now, stdout, stderr)
}

// decisionLogPath is <orchestrator.dir>/<project-id>/decisions.jsonl, with the
// project resolved as POST /v1/projects resolves it and the directory read
// through the configuration the daemon reads.
func decisionLogPath(ctx context.Context, project string, env func(string) string) (string, error) {
	ws, err := resolveWorkspace(project)
	if err != nil {
		return "", err
	}
	root, id, err := orchestrate.ResolveProject(ctx, ws)
	if err != nil {
		return "", err
	}
	resolved, err := app.Resolve(env, root)
	if err != nil {
		return "", err
	}
	dir, err := orchestrate.Dir(resolved, env)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id, "decisions.jsonl"), nil
}

// printDecisions prints every line the query keeps, and says on stderr every
// line that does not read and every seq that does not follow. Any of them
// makes the command fail once the rest is printed.
func printDecisions(ctx context.Context, path string, r *orchestrate.LogReader, q decisionsQuery,
	now func() time.Time, stdout, stderr io.Writer) error {
	problems := 0
	last := 0
	for {
		line, err := r.Next()
		if errors.Is(err, io.EOF) {
			if !q.follow {
				break
			}
			select {
			case <-ctx.Done():
				return problemsError(problems)
			case <-time.After(followEvery):
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		last = line.N
		if line.Err != nil {
			fmt.Fprintf(stderr, "dcode: %s:%d: %v\n", path, line.N, line.Err)
			problems++
			continue
		}
		if line.Gap {
			fmt.Fprintf(stderr, "dcode: %s:%d: seq %d follows %d; the log has a gap or a repetition\n",
				path, line.N, line.Decision.Seq, line.Prev)
			problems++
		}
		if !q.keeps(line.Decision) {
			continue
		}
		if q.json {
			fmt.Fprintf(stdout, "%s\n", line.Raw)
		} else {
			fmt.Fprintln(stdout, decisionText(line.Decision, now()))
		}
	}
	if p := r.Pending(); p != nil {
		fmt.Fprintf(stderr, "dcode: %s:%d: no newline at the end (%d bytes): a write in progress or one a crash cut short; not printed\n",
			path, last+1, len(p))
		problems++
	}
	return problemsError(problems)
}

func problemsError(n int) error {
	if n == 0 {
		return nil
	}
	return fmt.Errorf("the decision log has %d problem(s), said above", n)
}

func (q decisionsQuery) keeps(d orchestrate.Decision) bool {
	if q.child != "" && d.Child != q.child {
		return false
	}
	if q.kinds != nil && !q.kinds[d.Kind] {
		return false
	}
	return q.since.IsZero() || !d.At.Before(q.since)
}

// decisionText is one decision on one line:
// 21:04:05  #42  gate.measured  c1  red  make check (exit 2)  why: …
func decisionText(d orchestrate.Decision, now time.Time) string {
	at := d.At.Local()
	stamp := at.Format("15:04:05")
	if y, m, day := at.Date(); y != now.Year() || m != now.Month() || day != now.Day() {
		stamp = at.Format("2006-01-02 15:04:05")
	}
	parts := []string{stamp, fmt.Sprintf("#%d", d.Seq), d.Kind}
	if d.Child != "" {
		parts = append(parts, d.Child)
	}
	if m := measuredText(d.Measured); m != "" {
		parts = append(parts, m)
	}
	if d.Claim != "" {
		parts = append(parts, "claim: "+oneLine(d.Claim))
	}
	parts = append(parts, "why: "+oneLine(d.Why))
	return strings.Join(parts, "  ")
}

// measuredText is the measurement in a few words: a gate's result and
// command, a verdict, main's health.
func measuredText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == '"' {
		return "measured: cut, see --json"
	}
	var m struct {
		Result   string `json:"result"`
		Command  string `json:"command"`
		ExitCode *int   `json:"exit_code"`
		Verdict  string `json:"verdict"`
		Health   string `json:"health"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return "measured: unreadable, see --json"
	}
	switch {
	case m.Result != "":
		s := m.Result
		if m.Command != "" {
			s += "  " + m.Command
			if m.ExitCode != nil {
				s += fmt.Sprintf(" (exit %d)", *m.ExitCode)
			}
		}
		return s
	case m.Verdict != "":
		return m.Verdict
	case m.Health != "":
		return "main " + m.Health
	}
	return ""
}

func oneLine(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", " ⏎ ")
}
