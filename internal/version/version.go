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

// Injected via -ldflags at release time.
var (
	Version = ""
	Commit  = ""
	Date    = ""
)

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
	fmt.Fprintf(&b, "\n%s/%s, %s", runtime.GOOS, runtime.GOARCH, runtime.Version())
	return b.String()
}
