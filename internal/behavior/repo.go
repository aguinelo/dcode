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
