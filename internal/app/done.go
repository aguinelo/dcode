package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/loop/loopcommand"
	"github.com/aguinelo/dcode/internal/loop/qualifier"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/sandbox"
	"github.com/aguinelo/dcode/internal/tools"
)

// DoneFileName is where the definition of done is declared.
//
// Under .dcode/, which DefaultRules already submits to write confirmation. An
// agent that can edit its own definition of done widens its own reach, and that
// is literally why that rule exists — so this needs no new policy, only the
// right location.
const DoneFileName = "done.toml"

// loadDoneSet reads the definition of done for a workspace.
//
// The file is the same strict TOML subset the rest of the configuration uses:
//
//	protected = ["**/*_test.go"]
//
//	[tests]
//	command = "make test"
//
//	[lint]
//	command = "make lint"
//	exit_code = 0
//
// A verify command configured without a file is a set of exactly one. The two
// are not separate mechanisms — that is the whole point of the generalisation.
func loadDoneSet(path, verifyCommand string) (loop.DoneSet, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return doneFromVerify(verifyCommand), nil
	}
	if err != nil {
		return loop.DoneSet{}, err
	}

	sections, err := config.ParseSections(string(raw), path)
	if err != nil {
		return loop.DoneSet{}, err
	}

	var set loop.DoneSet
	for _, name := range sections.Order {
		values := sections.Values[name]
		if name == "" {
			if p := values["protected"]; p != "" {
				set.Protected = splitList(p)
			}
			continue
		}
		c := loop.Criterion{Name: name, Command: values["command"]}
		if v := values["exit_code"]; v != "" {
			c.ExitCode = atoi(v)
		}
		set.Criteria = append(set.Criteria, c)
	}

	if len(set.Criteria) == 0 {
		return doneFromVerify(verifyCommand), nil
	}
	return set, nil
}

func doneFromVerify(command string) loop.DoneSet {
	if strings.TrimSpace(command) == "" {
		return loop.DoneSet{}
	}
	return loop.DoneSet{Criteria: []loop.Criterion{{Name: "verify", Command: command}}}
}

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func atoi(v string) int {
	n := 0
	neg := false
	for i, r := range strings.TrimSpace(v) {
		if i == 0 && r == '-' {
			neg = true
			continue
		}
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	if neg {
		return -n
	}
	return n
}

// doneFilePath is where the definition lives for this workspace.
func doneFilePath(override, workspace string) string {
	if override != "" {
		return override
	}
	return filepath.Join(workspace, ".dcode", DoneFileName)
}

// parseDuration reads a duration setting, falling back rather than failing.
//
// The value comes from a file a person typed. Refusing to start a session over
// a malformed timeout is a worse answer than using the default and carrying on.
func parseDuration(v string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil || d <= 0 {
		return def
	}
	return d
}

// criterionRunner runs a done criterion through the sandbox.
//
// Through the sandbox, not around it. A criterion command is still a command:
// it comes from configuration a person reviewed, which is why it may run at all
// (RN-6.1 of configuration forbids running one read from a shared instruction
// file), but "reviewed" is not "unconfined".
func criterionRunner(sb sandbox.Sandbox, opts Options) loop.CriterionRunner {
	// Fixed, and deliberately so. A criterion is the daemon checking its own
	// definition of done, not the session doing work — it runs under the
	// boundary the session was configured with, whatever the person has since
	// switched the session to.
	runner := sandbox.Runner{Sandbox: sb, Mode: sandbox.Fixed(opts.SandboxMode)}
	return func(ctx context.Context, command string) (int, string, error) {
		out, code, err := runner.Run(ctx, opts.Workspace, command)
		return code, out, err
	}
}

// sessionDoneSet reads the definition of done a session is born with.
//
// A spec path wins over done.toml when one was given, and it is an explicit
// choice every time — never something SourceAuto falls into. Asking for a spec
// and silently getting the workspace's own done.toml would be the worst of the
// two: the turn measured against something the person did not name.
func sessionDoneSet(opts Options) (loop.DoneSet, error) {
	if opts.Qualify {
		// A qualifying session is not measured against anything: it is the one
		// working out what the measurement will be. Giving it a definition of
		// done would be asking it to satisfy the ruler it is drawing.
		return loop.DoneSet{}, nil
	}
	if strings.TrimSpace(opts.LoopSpec) == "" {
		return loadDoneSet(doneFilePath(opts.DoneFile, opts.Workspace), opts.VerifyCommand)
	}
	spec, err := loopcommand.LoadSpecWithProtect(opts.LoopSpec, opts.Protect)
	if err != nil {
		// Coded at the source. Left bare it reached the server's fallback,
		// which classified it by looking for the word "workspace" in the
		// message — and the message carries a path. Anyone whose project lives
		// under a directory called `workspace` got "workspace_invalid" for a
		// spec that simply was not there.
		return loop.DoneSet{}, protocol.Errorf(protocol.CodeInvalidInput, "%s", err.Error())
	}
	return spec.DoneSet(), nil
}

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
}

// Proposals is where a qualifying session keeps what the model proposed.
//
// One slot, replaced. A model that proposes twice has changed its mind, and
// keeping both would leave the loop choosing between them.
type Proposals struct {
	mu      sync.Mutex
	current *Proposal
}

// Take removes the recorded proposal and returns it. Nil when there is none.
//
// Take rather than Get: committing it is the end of its life, and a proposal
// that survived being written would be written again by the next commit.
func (p *Proposals) Take() *Proposal {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.current
	p.current = nil
	return out
}

func (p *Proposals) put(v *Proposal) {
	p.mu.Lock()
	p.current = v
	p.mu.Unlock()
}

// QualifyingTool builds the done_propose tool for a qualifying turn.
//
// The turn runs in PLAN mode: read-only, and no approval to grant. The tool
// touches nothing — it records, and the loop measures and writes afterwards.
func QualifyingTool(specDir string, into *Proposals) tools.DonePropose {
	return tools.DonePropose{
		Spec: specDir,
		Submit: func(_ context.Context, in tools.DoneProposeInput) (string, error) {
			p := &Proposal{Spec: specDir, Protected: in.Protected}
			for _, c := range in.Criteria {
				p.Criteria = append(p.Criteria, qualifier.Proposed{
					Name:     strings.TrimSpace(c.Name),
					Command:  strings.TrimSpace(c.Command),
					ExitCode: c.ExitCode,
					Expects:  qualifier.Expectation(strings.TrimSpace(c.Expects)),
					Why:      strings.TrimSpace(c.Why),
				})
			}
			into.put(p)
			return fmt.Sprintf("recorded %d criteria for %s.\n\n"+
				"They are NOT measured or written yet: this turn cannot write, and the "+
				"loop runs them and records the result once it ends. You are done — "+
				"do not start the work, and do not propose again unless one of these "+
				"is wrong.", len(p.Criteria), specDir), nil
		},
	}
}

// CommitProposal measures a recorded proposal and writes it into the spec
// folder, returning what a person reads.
//
// This is the loop's half, and it runs OUTSIDE the qualifying turn on purpose.
// Measuring under read-only would call a criterion broken because the sandbox
// refused it a cache directory, and a proposal born with a false measurement
// is worse than none.
func CommitProposal(ctx context.Context, p *Proposal, run loop.CriterionRunner, timeout time.Duration) (string, error) {
	if p == nil {
		return "", fmt.Errorf("app: nothing was proposed")
	}
	measured, cond, err := qualifier.Measure(ctx, qualifier.Proposal{
		Criteria: p.Criteria, Protected: p.Protected,
	}, run, timeout)
	if err != nil {
		return "", err
	}
	path := filepath.Join(p.Spec, tools.DoneProposeFile)
	if err := os.WriteFile(path, qualifier.Render(measured, p.Protected, cond), 0o644); err != nil {
		return "", fmt.Errorf("could not write %s: %w", path, err)
	}
	return qualifier.Summary(measured, cond, path), nil
}
