// Package version reports what build this is.
//
// The values are injected at link time. A build without them reports "dev"
// rather than an empty string or a plausible-looking version: a binary that
// cannot say what it is should say so, not guess.
package version

import (
	"fmt"
	"runtime"
	"strings"
)

// Injected via -ldflags at build time.
var (
	Version = ""
	Commit  = ""
	Date    = ""
	// Source is how this binary was produced. Only the release pipeline sets
	// it to SourceRelease; every other path leaves it as it found it.
	//
	// It is a separate variable rather than a shape inferred from Version
	// because `update` refuses to overwrite a binary it did not publish, and a
	// decision like that should not rest on parsing a string that anyone can
	// pass on a command line.
	Source = ""
)

// The two origins that matter. Anything else is a build we cannot account for,
// and is treated as local — the cautious reading, since it is the one that
// refuses to overwrite.
const (
	SourceRelease = "release"
	SourceLocal   = "local"
)

// IsRelease reports whether the release pipeline produced this binary.
func IsRelease() bool { return Source == SourceRelease }

// Short returns the version alone.
func Short() string {
	if Version == "" {
		return "dev"
	}
	return Version
}

// String returns version, commit, date and platform.
func String() string {
	var b strings.Builder
	b.WriteString("dcode ")
	b.WriteString(Short())

	if Commit != "" {
		c := Commit
		if len(c) > 12 {
			c = c[:12]
		}
		fmt.Fprintf(&b, " (%s)", c)
	}
	if Date != "" {
		fmt.Fprintf(&b, " built %s", Date)
	}
	// A local build says so. A binary that presents itself exactly like a
	// published release is how a bug report costs an hour finding out it was
	// never the published code.
	if !IsRelease() {
		b.WriteString(" · local build")
	}
	fmt.Fprintf(&b, "\n%s/%s, %s", runtime.GOOS, runtime.GOARCH, runtime.Version())
	return b.String()
}
