package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/session"
)

// runSessions lists what dcode has done and prints one back.
//
// Reading only. Everything here answers a question about the past — what was I
// doing, what did it actually do — without a model, a daemon or a turn, which
// is why it exists before continuing a session does.
func runSessions(args []string) error {
	fs := flag.NewFlagSet("dcode sessions", flag.ContinueOnError)
	all := fs.Bool("all", false, "every workspace, not only this one")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `usage:
  dcode sessions [--all]        list recorded sessions
  dcode sessions show <id>      print one as the conversation it was

`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) > 0 && rest[0] == "show" {
		if len(rest) < 2 {
			return fmt.Errorf("which session? try `dcode sessions` to see them")
		}
		return showSession(rest[1])
	}
	if len(rest) > 0 {
		return fmt.Errorf("unknown argument %q; try `dcode sessions` or `dcode sessions show <id>`", rest[0])
	}
	return listSessions(*all)
}

func listSessions(all bool) error {
	dir, err := sessionDir()
	if err != nil {
		return err
	}
	filter := ""
	if !all {
		// "What was I doing here" is the question nearly every time. The other
		// one gets a flag rather than the default.
		if filter, err = os.Getwd(); err != nil {
			return err
		}
	}

	found, err := session.Browse(dir, filter)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		if all {
			fmt.Println("no sessions recorded yet")
			return nil
		}
		fmt.Println("no sessions recorded for this workspace — `dcode sessions --all` for the rest")
		return nil
	}

	for _, s := range found {
		line := fmt.Sprintf("%-12s  %s  %2d turn(s)  %s",
			s.ID, s.Started.Local().Format("2006-01-02 15:04"), s.Turns, s.Title)
		if all {
			line += "  " + s.Workspace
		}
		fmt.Println(line)
	}
	return nil
}

func showSession(id string) error {
	dir, err := sessionDir()
	if err != nil {
		return err
	}
	// The id is a filename, so it must not be able to name one outside the
	// directory. `dcode sessions show ../../etc/passwd` is the shape of that
	// mistake, and rejecting it here is cheaper than trusting the caller.
	if id != filepath.Base(id) || strings.HasPrefix(id, ".") {
		return fmt.Errorf("%q is not a session id", id)
	}

	body, err := session.Transcript(filepath.Join(dir, id+".jsonl"))
	if os.IsNotExist(err) {
		return fmt.Errorf("no session %s here; try `dcode sessions`", id)
	}
	if err != nil {
		return err
	}
	fmt.Print(body)
	return nil
}

// sessionDir resolves where records live, through the same configuration the
// daemon reads. Two places deciding it is two places to get it wrong.
func sessionDir() (string, error) {
	ws, err := os.Getwd()
	if err != nil {
		return "", err
	}
	resolved, err := app.Resolve(os.Getenv, ws)
	if err != nil {
		return "", err
	}
	return recordDir(resolved), nil
}
