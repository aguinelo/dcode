package sandbox

import (
	"path/filepath"
	"runtime"
)

// goos is the platform, injectable so both branches below are reachable from a
// test. A branch that only one platform can execute is a branch only one
// platform can check, and this one decides where a compiler is allowed to
// write.
var goos = runtime.GOOS

// Scratch are the directories outside the workspace that a toolchain must write
// to in order to build at all.
//
// They exist because a compiler's cache is shared across projects and far too
// large to keep inside one: Go, Rust, Node and Java all put theirs under the
// user's home. Without them `go test` cannot run at all — measured on this
// repository, where an unattended session changed files and then could not
// execute a single test, the first failure being
// `open ~/Library/Caches/go-build/…: operation not permitted`.
//
// A sandbox that permits editing and forbids checking produces unverified
// change, which is the outcome the definition of done exists to prevent.
//
// Named directories, never the home itself. The list is the boundary: a rule
// reaching $HOME would hand over ssh keys along with the compiler's scratch
// space, and that is the difference between granting a cache and granting a
// person's machine.
//
// The environment is read where a toolchain publishes its own answer, because
// the machine with GOCACHE somewhere unusual is the one most likely to be
// denied. env is passed in rather than read here, like the language and the
// palette: this package builds profiles and does not consult the world.
func Scratch(env func(string) string) []string {
	// A nil lookup must neither crash nor open, which is the rule this package
	// already applies to a nil network decision: a session must not die over a
	// wiring mistake, and nothing said is never a yes.
	if env == nil {
		return nil
	}
	home := env("HOME")
	if home == "" {
		// Nothing to derive from, and nothing invented: a path built from an
		// empty string is "/.cache", which is not a cache and is somewhere
		// nobody meant to grant.
		return nil
	}

	add := func(out []string, configured string, fallback ...string) []string {
		if configured != "" {
			return append(out, configured)
		}
		if len(fallback) == 0 {
			// Nothing named it and there is no default worth guessing.
			return out
		}
		return append(out, filepath.Join(append([]string{home}, fallback...)...))
	}

	var out []string
	// The per-user temporary directory, which on macOS is not /tmp: it is
	// $TMPDIR under /var/folders, and the Go toolchain stages every
	// compilation there. Granting /tmp and /var/tmp and stopping short of this
	// one was the second blocker found — the first was the build cache below,
	// and fixing only that moved the failure rather than removing it.
	out = add(out, env("TMPDIR"))
	// Go: the build cache is written on every compile, the module cache on
	// every dependency resolution.
	if goos == "darwin" {
		out = add(out, env("GOCACHE"), "Library", "Caches", "go-build")
	} else {
		out = add(out, env("GOCACHE"), ".cache", "go-build")
	}
	out = add(out, env("GOMODCACHE"), "go", "pkg", "mod")
	// Rust, Node, Java, Python: same shape, different names.
	out = add(out, env("CARGO_HOME"), ".cargo")
	out = add(out, env("npm_config_cache"), ".npm")
	out = add(out, "", ".m2")
	// The generic one, last, because a toolchain that respects it will already
	// have been named above.
	out = add(out, env("XDG_CACHE_HOME"), ".cache")

	return dedupe(out, home)
}

// dedupe drops repeats and refuses the home directory itself, whatever an
// environment variable says. A cache path that resolves to $HOME is a
// misconfiguration, and honouring it would grant everything.
func dedupe(paths []string, home string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		if p == "" || p == home || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
