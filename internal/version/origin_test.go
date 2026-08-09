package version

import (
	"strings"
	"testing"
)

// The origin is a separate variable rather than a shape inferred from the
// version string, because `update` refuses to overwrite a binary it did not
// publish and a decision like that should not rest on parsing.
func TestIsReleaseOnlyForTheReleasePipeline(t *testing.T) {
	restore := func(s, v string) func() {
		os, ov := Source, Version
		Source, Version = s, v
		return func() { Source, Version = os, ov }
	}

	for name, tc := range map[string]struct {
		source string
		want   bool
	}{
		"the release pipeline": {SourceRelease, true},
		"make install":         {SourceLocal, false},
		// go install, or anything else we cannot account for. Treated as local
		// because that is the reading that refuses to overwrite.
		"nothing injected": {"", false},
		"something else":   {"whatever", false},
	} {
		t.Run(name, func(t *testing.T) {
			defer restore(tc.source, "1.2.3")()
			if got := IsRelease(); got != tc.want {
				t.Errorf("got %v", got)
			}
		})
	}
}

// A binary that presents itself exactly like a published release is how a bug
// report costs an hour finding out it was never the published code.
func TestALocalBuildSaysSoAndAReleaseDoesNot(t *testing.T) {
	oldSource, oldVersion := Source, Version
	defer func() { Source, Version = oldSource, oldVersion }()

	Source, Version = SourceLocal, "0.0.0-dev+a91f2c4"
	local := String()
	if !strings.Contains(local, "local build") {
		t.Errorf("a local build must say so:\n%s", local)
	}
	if !strings.Contains(local, "0.0.0-dev+a91f2c4") {
		t.Errorf("and still report what it is:\n%s", local)
	}

	Source, Version = SourceRelease, "0.1.0"
	released := String()
	if strings.Contains(released, "local build") {
		t.Errorf("a published release must not:\n%s", released)
	}
}

// A build with nothing injected still has to say what it is rather than guess.
func TestAnUnstampedBuildReportsDev(t *testing.T) {
	oldSource, oldVersion := Source, Version
	defer func() { Source, Version = oldSource, oldVersion }()

	Source, Version = "", ""
	if Short() != "dev" {
		t.Errorf("got %q", Short())
	}
	if !strings.Contains(String(), "local build") {
		t.Errorf("an unaccountable build is treated as local: %s", String())
	}
}
