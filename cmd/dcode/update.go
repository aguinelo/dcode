package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/update"
	"github.com/aguinelo/dcode/internal/version"
)

// runUpdate installs a newer release, on request and never otherwise.
func runUpdate(args []string) error {
	fs := flag.NewFlagSet("dcode update", flag.ContinueOnError)
	var (
		check   = fs.Bool("check", false, "report whether a newer version exists and exit")
		channel = fs.String("channel", "", "stable or prerelease (default: $DCODE_RELEASE_CHANNEL, else stable)")
		force   = fs.Bool("force", false, "replace a local build with a published release, even if that is older")
	)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"dcode update — install the latest release\n\n"+
				"The signature and the checksum are both verified; either failing aborts\n"+
				"and leaves the current binary untouched.\n\n"+
				"A local build is refused: it is normally ahead of the last tag, so\n"+
				"replacing it would be a downgrade. Use `make install` to rebuild.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	ch := *channel
	if ch == "" {
		ch = os.Getenv("DCODE_RELEASE_CHANNEL")
	}

	u := update.NewGitHub(update.Config{
		APIURL:              os.Getenv("DCODE_UPDATE_URL"),
		Channel:             ch,
		Pin:                 os.Getenv("DCODE_PIN_VERSION"),
		AllowLocalOverwrite: *force,
	})

	// Before the network, not after: if the answer is a refusal either way,
	// asking GitHub first spends a round trip to report the wrong reason —
	// "no release found" when the real answer is "this binary is not ours to
	// replace". Apply refuses again, which is where the guarantee lives.
	if !*check && !*force && !version.IsRelease() {
		return update.ErrLocalBuild
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rel, err := u.Latest(ctx)
	if err != nil {
		return err
	}

	current := version.Short()
	if *check {
		n := update.VersionNotice{Current: current, Latest: rel.Version, CheckedAt: time.Now()}
		if msg := n.Message(); msg != "" {
			fmt.Println(msg)
			return nil
		}
		fmt.Printf("dcode %s is the latest release.\n", current)
		return nil
	}

	if (update.VersionNotice{Current: current, Latest: rel.Version}).Message() == "" {
		fmt.Printf("dcode %s is already the latest release.\n", current)
		return nil
	}

	fmt.Printf("Updating dcode %s → %s\n", current, rel.Version)
	if err := u.Apply(ctx, rel); err != nil {
		return err
	}
	fmt.Printf("Updated to %s.\n", rel.Version)
	return nil
}

// noticePath is where the passive check caches its answer. Under the state
// root, never under config: it is derived data with its own lifecycle.
func noticePath(env func(string) string) (string, error) {
	roots, err := config.DiscoverRoots(env)
	if err != nil {
		return "", err
	}
	return filepath.Join(roots.State, update.NoticeFileName), nil
}

// versionNotice returns the line a client shows, or empty.
//
// It never blocks anything and never fails: a network problem returns whatever
// was cached, which may be nothing at all.
func versionNotice(ctx context.Context) string {
	if !update.ParseBool(os.Getenv("DCODE_UPDATE_CHECK"), true) {
		return ""
	}
	path, err := noticePath(os.Getenv)
	if err != nil {
		return ""
	}
	u := update.NewGitHub(update.Config{
		APIURL:  os.Getenv("DCODE_UPDATE_URL"),
		Channel: os.Getenv("DCODE_RELEASE_CHANNEL"),
	})
	interval := update.ParseInterval(os.Getenv("DCODE_UPDATE_CHECK_INTERVAL"))
	n := update.Check(ctx, u, path, version.Short(), time.Now(), interval)
	return n.Message()
}
