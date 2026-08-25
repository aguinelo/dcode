package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/tui"
	"github.com/aguinelo/dcode/internal/update"
	"github.com/aguinelo/dcode/internal/version"
)

// runUpdate installs a newer release, on request and never otherwise.
func runUpdate(args []string) error {
	fs := flag.NewFlagSet("dcode update", flag.ContinueOnError)
	var (
		check   = fs.Bool("check", false, "report whether a newer version exists and exit")
		channel = fs.String("channel", "", "stable or prerelease (default: update.channel or $DCODE_UPDATE_CHANNEL, else stable)")
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

	// The flag wins, then the ordinary configuration chain — which is what
	// makes `update.channel` in a config file reach this command at all.
	ch := *channel
	if ch == "" {
		ch = updateSetting("update.channel", "")
	}
	if ch == "" {
		ch = legacyChannel()
	}

	u := update.NewGitHub(update.Config{
		APIURL: os.Getenv("DCODE_UPDATE_URL"),
		// The second route to the expected digest. An internal mirror that
		// overrides DCODE_UPDATE_URL has to override this too, or the only
		// copy of the digest that did not travel with the artifact still comes
		// from upstream.
		InstallerURL:        os.Getenv("DCODE_UPDATE_INSTALLER_URL"),
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
	if err := replaceBinary(ctx, u, rel); err != nil {
		return err
	}
	fmt.Printf("Updated to %s.\n", rel.Version)
	return nil
}

// replaceBinary is the ONE place this binary is replaced.
//
// Both ways of asking — `dcode update` and `/update` from inside the client —
// come through here, and a test asserts that `Apply` is called from exactly one
// place. The rule it protects is not "one entry point": it is that no path
// replaces the binary without having been asked to. Two doors would be two
// things to check every time either changes.
func replaceBinary(ctx context.Context, u *update.GitHub, rel update.Release) error {
	return u.Apply(ctx, rel)
}

// updater is what `/update` calls from inside the client.
//
// The same updater the command builds, with the same refusals: a local build is
// not replaced, a pin is honoured, and going backwards is not an update. What
// it does NOT do is restart — replacing the binary under a running process
// leaves the running process being the old one, and the client says so rather
// than implying otherwise.
func updater(ctx context.Context) (tui.UpdateResult, error) {
	ch := updateSetting("update.channel", "")
	if ch == "" {
		ch = legacyChannel()
	}
	u := update.NewGitHub(update.Config{
		APIURL:       os.Getenv("DCODE_UPDATE_URL"),
		InstallerURL: os.Getenv("DCODE_UPDATE_INSTALLER_URL"),
		Channel:      ch,
		Pin:          os.Getenv("DCODE_PIN_VERSION"),
	})

	current := version.Short()
	// Before the network, for the reason runUpdate gives: if the answer is a
	// refusal either way, asking GitHub first spends a round trip to report the
	// wrong reason.
	if !version.IsRelease() {
		return tui.UpdateResult{}, update.ErrLocalBuild
	}
	rel, err := u.Latest(ctx)
	if err != nil {
		return tui.UpdateResult{}, err
	}
	if (update.VersionNotice{Current: current, Latest: rel.Version}).Message() == "" {
		return tui.UpdateResult{From: current}, nil
	}
	if err := replaceBinary(ctx, u, rel); err != nil {
		return tui.UpdateResult{}, err
	}
	return tui.UpdateResult{From: current, To: rel.Version, Applied: true}, nil
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
	if !update.ParseBool(updateSetting("update.check", "true"), true) {
		return ""
	}
	path, err := noticePath(os.Getenv)
	if err != nil {
		return ""
	}
	u := update.NewGitHub(update.Config{
		APIURL:  os.Getenv("DCODE_UPDATE_URL"),
		Channel: cmp.Or(updateSetting("update.channel", ""), legacyChannel()),
	})
	interval := update.ParseInterval(os.Getenv("DCODE_UPDATE_CHECK_INTERVAL"))
	n := update.Check(ctx, u, path, version.Short(), time.Now(), interval)
	return n.Message()
}

// updateSetting reads one key through the ordinary configuration chain.
//
// The update paths are not a session, so they have no resolved Options to read
// from. Reading os.Getenv directly instead is what made `update.channel` a
// declared key that no config file could ever reach: the key mapped to one
// variable name and the code read another, and nothing crossed the two.
//
// A failure here is not worth reporting — configuration that cannot be
// resolved means there is no workspace to speak of, and the update path must
// never be what blocks the binary from running.
func updateSetting(key, def string) string {
	wd, err := os.Getwd()
	if err != nil {
		return def
	}
	r, err := app.Resolve(os.Getenv, wd)
	if err != nil {
		return def
	}
	return r.String(key, def)
}

// legacyChannel reads the variable this command used before `update.channel`
// reached it through configuration.
//
// Kept as a fallback rather than removed: it was the only spelling that
// worked, so it is the one people who set a channel at all are using, and
// dropping it would move them back to stable without a word.
func legacyChannel() string { return os.Getenv("DCODE_RELEASE_CHANNEL") }
