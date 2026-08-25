package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/sandbox"
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
