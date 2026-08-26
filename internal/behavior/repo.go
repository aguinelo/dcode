package behavior

import (
	"fmt"
	"strings"
)

// Repo is where the agent is working, as it stood when the session opened.
//
// Data, never a reader: Build is pure and must stay pure, so nothing here runs
// a command. Whoever creates the session takes the snapshot and passes it in,
// the same way the instruction chain is frozen and handed over.
//
// It is deliberately not a tool. `bash` already runs git, and a tool the model
// has to remember to call is a fact it uses when it happens to think of it. The
// branch it is on is not that kind of fact: every rule about where work belongs
// depends on it, and a rule that needs a lookup first is a rule followed by
// accident.
type Repo struct {
	// Absent marks a workspace that is not a repository at all.
	//
	// It used to be nil, and nil rendered nothing. The comment above the field
	// said that was "ordinary and silent" and the invariant said the prefix
	// carries "nothing when it is not". Ordinary, yes. Silent, no.
	//
	// An agent worked for a day in a directory with no repository: 191 lines of
	// code, 35 spec files, and a project file of its own writing that demanded
	// a commit per task, a pull request per spec and a coverage floor "for
	// merging into main". None of it could happen and nothing said so. The
	// absence is not a detail of the environment — it decides what finishing
	// the work even means, because there is no diff for anyone to read, no
	// review, and no undo that is not rewriting the file by hand.
	//
	// Nil still means the snapshot was not taken. "We did not look" and "we
	// looked and there is none" are different facts, and only one of them is
	// worth a line.
	Absent bool
	// Branch is empty when there is none to name — a detached head, or a
	// repository with no commit yet.
	Branch string
	// MainBranch is what work is normally cut from and merged back into. It is
	// what "am I on the right branch" is answered against.
	MainBranch string
	// Detached marks a head that is not on a branch. Separate from an empty
	// Branch because "no branch" and "we did not find out" are different, and
	// only one of them is worth telling the model.
	Detached bool
	// Clean is stated rather than derived from an empty Status: "nothing
	// changed" and "we did not look" read the same when both are blank.
	Clean bool
	// Status is porcelain output, already bounded by whoever read it.
	Status string
	// Commits are the most recent, newest first, one line each.
	Commits []string
	// Truncated marks a status that did not fit. Nothing in this codebase cuts
	// output without saying so.
	Truncated bool
}

// renderRepo writes what the prefix says about the repository, or "" when there
// is nothing to say.
//
// Every line is a fact the model would otherwise have to go and ask for, and
// the snapshot is labelled as one. The repository moves while the session runs
// — the agent's own commits move it — so presenting this as current would be a
// statement that was true at the start and quietly false after the first
// commit.
func renderRepo(r *Repo) string {
	if r == nil {
		return ""
	}
	if r.Absent {
		// A fact, not a warning, and not a thing to ask permission about. The
		// work goes ahead either way; what changes is that nobody finds out
		// afterwards. Saying it once is the whole of it — repeating it every
		// turn would be the nagging this deliberately is not.
		return "This workspace is NOT a git repository.\n\n" +
			"Nothing written here has history: there is no diff to review, no undo " +
			"beyond rewriting a file by hand, and no commit, branch or pull request " +
			"is possible. Any instruction that asks for one — yours, the project's, " +
			"or your own plan's — cannot be carried out here.\n\n" +
			"Say this once, plainly, before you first write, and offer `git init`. " +
			"Then do the work as asked. Do not ask again, do not repeat it each turn, " +
			"and do not make it a condition of starting."
	}
	var b strings.Builder
	b.WriteString("A snapshot from when this session opened. It does not update, " +
		"and your own commits move it — run git yourself when the answer has to be current.\n")

	switch {
	case r.Detached:
		b.WriteString("\nHEAD is detached: not on a branch. Say so before committing anywhere.")
	case r.Branch != "":
		fmt.Fprintf(&b, "\nCurrent branch: %s", r.Branch)
	}
	if r.MainBranch != "" {
		fmt.Fprintf(&b, "\nMain branch: %s", r.MainBranch)
	}

	if r.Clean {
		b.WriteString("\n\nWorking tree: clean.")
	} else if strings.TrimSpace(r.Status) != "" {
		b.WriteString("\n\nUncommitted changes:\n")
		b.WriteString(strings.TrimRight(r.Status, "\n"))
		if r.Truncated {
			b.WriteString("\n… more, not shown.")
		}
	}

	if len(r.Commits) > 0 {
		b.WriteString("\n\nRecent commits:\n")
		for i, c := range r.Commits {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(c)
		}
	}
	return b.String()
}

// Workspace is what the project declares about itself, as it stood when the
// session opened.
//
// Data, never a reader — the same rule as Repo, for the same reason: Build is
// pure and nothing here runs a command or touches a disk. Whoever creates the
// session probes and passes the result in.
//
// Gate mirrors workspace.Gate rather than importing it. behavior imports
// nothing that reads a disk, and a type alias would be an import.
type Workspace struct {
	// Gates are the commands the project declares as its own checks.
	Gates []Gate
	// Truncated marks a list that did not fit. Nothing in this codebase cuts
	// output without saying so.
	Truncated bool
}

// Gate is one declared check, at the prefix boundary.
type Gate struct {
	Name    string
	Command string
	Source  string
}

// renderWorkspace writes what the prefix says about the project's own checks,
// or "" when there is nothing to say.
//
// Nothing to say is the common case and it is silent, unlike an absent
// repository. The difference is consequence: having no repository changes what
// finishing the work means, while declaring no gate is ordinary and changes
// nothing. Not every absence earns a line — what must not happen is an absence
// nobody checked becoming a claim.
func renderWorkspace(w *Workspace) string {
	if w == nil || len(w.Gates) == 0 {
		return ""
	}
	// A name that two sources both declare is qualified by its source.
	//
	// Found by running it: a project with a `test` script and a `test` target
	// printed two rows called `test` with different commands, and nothing on
	// screen said which was which. That is the same defect the task parser
	// refuses in a `tasks.md` — two rows a reader cannot tell apart — and the
	// Source was being carried and never shown.
	//
	// Only the ambiguous ones. Qualifying every row would spend a column on a
	// distinction that almost never matters.
	count := map[string]int{}
	for _, g := range w.Gates {
		count[g.Name]++
	}
	label := func(g Gate) string {
		if count[g.Name] > 1 && g.Source != "" {
			return g.Name + " (" + g.Source + ")"
		}
		return g.Name
	}

	width := 0
	for _, g := range w.Gates {
		if n := len(label(g)); n > width {
			width = n
		}
	}

	var b strings.Builder
	b.WriteString("This project declares its own checks:\n")
	for _, g := range w.Gates {
		fmt.Fprintf(&b, "\n  %-*s  %s", width, label(g), g.Command)
	}
	if w.Truncated {
		b.WriteString("\n  … more, not shown.")
	}
	// The load-bearing sentence. Without it a list of gates reads as a list of
	// guarantees, and this whole section would have produced the defect that
	// asked for it: a project with four declared gates, two red since the first
	// day, and nobody knowing.
	b.WriteString("\n\nThese are what the project measures itself by. " +
		"Nothing here says they pass, and nothing has run them.")
	return b.String()
}
