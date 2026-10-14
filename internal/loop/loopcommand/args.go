package loopcommand

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The grammar of `/loop`, in the daemon.
//
// A copy of internal/tui's, moved here so the daemon that conducts a loop
// parses what a client sent as it was typed (desktop/docs/proposals/loop.md
// §4.1). The TUI keeps its own until it asks the daemon to run the loop; that
// pull request deletes the copy there, and this one is the home.

// ErrLoopUsage is `/loop` given nothing to work on.
var ErrLoopUsage = errors.New("usage: /loop <path to a spec folder> [what to do] [--protect <glob>], or /loop <goal>")

// LoopArgs is what `/loop` was given, after parsing.
type LoopArgs struct {
	// Task is what to do, in the person's own words, when they said.
	//
	// Everything after the path that is not a flag. `/loop specs/x` is the
	// command doing its job on its own; `/loop specs/x refaça só o header` is
	// the same job narrowed, and refusing the second was the command telling
	// someone their sentence was a mistyped flag.
	Task string
	// Spec is the folder holding tasks.md, as the user typed it. Resolving it
	// is the daemon's job: it owns the filesystem, and a client that resolved
	// the path would be asserting something about a disk it may not share.
	Spec string
	// Protect are globs added to whatever the spec declares.
	Protect []string
	// Qualify marks the session that works out what done means for Spec,
	// rather than the one that does the work.
	//
	// The LOOP sets it, never the model: reading, projecting and qualifying
	// are what the loop does before it executes, and a model that chose when
	// to qualify would be choosing when to be measured.
	Qualify bool
	// Goal marks an argument that is a sentence rather than a path.
	//
	// `/loop implemente todas as specs pendentes` is what someone types when
	// they mean the whole backlog, and the first version made `implemente`
	// into a folder name and then failed to read `implemente/tasks.md`. Prose
	// became a path — the same defect as prose becoming a criterion, in the
	// other direction.
	Goal bool
}

// FlagError is something that looks like a flag and is not one. It carries
// the token, so a client can say it in the person's language.
type FlagError struct{ Flag string }

func (e FlagError) Error() string { return e.Flag + " is not a flag here — only --protect is" }

// ParseLoopArgs splits the argument of `/loop`. Pure, and strict.
//
// Strict because a mistyped flag that is silently ignored produces a session
// measured against a definition of done the person did not ask for, and they
// find out at the end of the turn. An unknown flag stops the command.
func ParseLoopArgs(args string) (LoopArgs, error) {
	var out LoopArgs
	var words []string
	fields := strings.Fields(args)
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		switch {
		case f == "--protect":
			if i+1 >= len(fields) {
				return LoopArgs{}, fmt.Errorf("--protect needs a glob after it")
			}
			i++
			out.Protect = append(out.Protect, fields[i])
		case strings.HasPrefix(f, "--protect="):
			g := strings.TrimPrefix(f, "--protect=")
			if g == "" {
				return LoopArgs{}, fmt.Errorf("--protect needs a glob after it")
			}
			out.Protect = append(out.Protect, g)
		case strings.HasPrefix(f, "-"):
			// Only a thing that LOOKS like a flag can be a mistyped one. A word
			// is a word.
			return LoopArgs{}, FlagError{Flag: f}
		default:
			words = append(words, f)
		}
	}
	if len(words) == 0 {
		return LoopArgs{}, ErrLoopUsage
	}
	if specArgument(words) {
		out.Spec = words[0]
		out.Task = strings.Join(words[1:], " ")
		return out, nil
	}
	out.Goal = true
	out.Task = strings.Join(words, " ")
	return out, nil
}

// specArgument says the words name one spec folder rather than describing work
// across all of them.
//
// Deterministic, and it never touches a disk the client may not share. A first
// word carrying a separator is a path — `specs/home-page` is how a person
// writes one. A single word is a path too, because one word is what a folder
// name looks like and the error names it when it is not there. Anything else
// is a sentence.
//
// The first version had no rule at all: the first word was the path, always.
// So `/loop implemente todas as specs pendentes` went looking for
// `implemente/tasks.md`. Prose became a path, which is the same defect as
// prose becoming a criterion, pointing the other way.
func specArgument(words []string) bool {
	if len(words) == 1 {
		return true
	}
	return strings.ContainsAny(words[0], "/"+string(filepath.Separator))
}

// LoopTask is the turn `/loop` submits.
//
// It submits one at all because loading a definition of done and then waiting
// is the command doing half its job: `/loop specs/x` means "do this", and it
// used to mean "open a session against this and sit there". Someone typed it,
// watched nothing happen, and had to say what they wanted anyway.
//
// The criteria are NOT restated here. They are already the session's, the loop
// checks them, and a copy in the first message is a second statement of
// something that can move.
func LoopTask(spec LoopArgs) string {
	if spec.Qualify {
		// A different job, so a different instruction. This turn produces a
		// proposal and nothing else — it cannot write, and the tool is the
		// only way its answer reaches anybody.
		subject := spec.Spec
		read := "Read the specification, its tasks, and enough of the code to see what " +
			"already exists and what the project can actually run."
		if spec.Goal {
			// The sentence is the whole brief, so the sentence is what has to
			// be named. "Work out how .dcode will be known to be finished" is
			// the anchor talking rather than the request, and sending the model
			// to read a specification that does not exist spends its rounds
			// looking for one.
			subject = strconv.Quote(strings.TrimSpace(spec.Task))
			read = "No specification folder describes this: the sentence above is the " +
				"whole brief. Read enough of the code to see what already exists and " +
				"what the project can actually run."
		}
		return "Work out how " + subject + " will be known to be finished.\n\n" +
			read + " Then call `done_propose` with the criteria, as commands.\n\n" +
			"You are in plan mode: you cannot change anything, and you are not " +
			"meant to. Propose, and stop."
	}
	if spec.Task != "" {
		return spec.Task
	}
	return "Implement the specification in " + spec.Spec + ".\n\n" +
		"Read what is there first. This session already carries that folder's " +
		"definition of done, and the harness checks it — do not go looking for " +
		"the criteria to run them yourself, and do not report done on your own " +
		"word."
}

// GoalToQualify turns a goal that found no spec folders into the session that
// works out what its criteria are.
//
// `/loop revise o projeto até entender` used to answer "no specs/ folder here"
// — the command telling someone their request was the wrong shape, in a product
// whose done-qualifier family exists for exactly this. That family's own
// research spec names the prose request as what motivated it: "Faça um cadastro
// de clientes" carries no tasks.md, and the constructive answer is to raise the
// criteria, measure them, and ask for a signature.
//
// `/loop <path>` already does this when the folder declares nothing. Only the
// sentence route dead-ended, and it dead-ended because it was built to SELECT
// among folders rather than to accept a brief.
//
// The proposal is anchored at `.dcode`, where a workspace's definition of done
// already lives and which is one of the three sources a DoneSet is read from.
// Not a folder derived from the sentence: the loop-command spec records the
// opposite defect, prose becoming a path, when `/loop implemente todas as specs
// pendentes` went looking for `implemente/tasks.md`. Inventing
// `revise-o-projeto/` would be that defect wearing the other hat.
func GoalToQualify(spec LoopArgs, found []protocol.SpecFolder) (LoopArgs, bool) {
	if !spec.Goal || len(found) > 0 || strings.TrimSpace(spec.Task) == "" {
		return LoopArgs{}, false
	}
	spec.Qualify = true
	spec.Spec = protocol.WorkspaceDoneDir
	return spec, true
}

// ArgumentToQualify turns a `/loop <word>` that names no folder into a goal.
//
// specArgument reads a single word as a path, because "one word is what a
// folder name looks like and the error names it when it is not there". The
// error it named was the daemon's own:
//
//	could not open a session: invalid_input: loopcommand: read
//	/Users/…/oi/tasks.md: open /Users/…/oi/tasks.md: no such file or directory
//
// The absolute path twice, for someone who typed a word. And the rule that a
// goal with no folder gets qualified never reached it, because a bare word
// never became a goal at all.
//
// The separator is what decides. Someone who wrote `specs/hoem` meant a path,
// and a typo answered by qualifying it as a goal is a typo hidden — so that one
// stays an error. A bare word that names nothing is prose, and prose is what
// the qualifier is for.
func ArgumentToQualify(spec LoopArgs, found []protocol.SpecFolder) (LoopArgs, bool) {
	if spec.Goal || spec.Spec == "" || spec.Qualify {
		return LoopArgs{}, false
	}
	if strings.ContainsAny(spec.Spec, "/"+string(filepath.Separator)) {
		return LoopArgs{}, false
	}
	for _, f := range found {
		if f.Path == spec.Spec {
			return LoopArgs{}, false
		}
	}
	// The word is the brief, and anything typed after it belongs to it.
	spec.Task = strings.TrimSpace(spec.Spec + " " + spec.Task)
	spec.Goal = true
	return GoalToQualify(spec, nil)
}
