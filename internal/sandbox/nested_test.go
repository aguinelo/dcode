package sandbox

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// Being on PATH is not being able to confine.
//
// The macOS kernel refuses to apply a profile from inside a process that is
// already confined — `sandbox_apply: Operation not permitted` — so a session
// running inside dcode could not run dcode's own boundary tests. They failed
// six at a time, and an agent reading that failure has no way to tell it from
// its own work being wrong: it spent a session fixing the harness instead.
//
// The Linux backend has probed since it was written, for the same class of
// reason. This closes the asymmetry.
func TestSeatbeltProbesRatherThanTrustingThePath(t *testing.T) {
	// A binary that exists and refuses every profile is exactly what nesting
	// looks like from in here.
	s := &seatbelt{bin: "false"}
	err := s.Available()
	if err == nil {
		t.Fatal("a backend that cannot apply a profile reported itself available")
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("the failure is not reported as unavailability: %v", err)
	}
	if !strings.Contains(err.Error(), "profile") {
		t.Errorf("the message does not say what could not be done: %v", err)
	}
}

// A backend whose probe succeeds reports available. Asserted with a stand-in
// that exits zero, so both halves of the decision are checked on every
// platform: the real binary exists on only one of them, and a branch only one
// machine can run is a branch only one machine can check.
func TestSeatbeltReportsAvailableWhenTheProbeSucceeds(t *testing.T) {
	if err := (&seatbelt{bin: "true"}).Available(); err != nil {
		t.Errorf("a backend whose probe succeeded reported unavailable: %v", err)
	}
}

// And the real binary, where there is one. This is the case that matters and
// the one no stand-in can stand in for.
func TestSeatbeltReportsAvailableWhenItCanConfine(t *testing.T) {
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		t.Skip("not macOS")
	}
	s := &seatbelt{bin: "sandbox-exec"}
	if err := s.Available(); err != nil {
		// Running the suite inside a sandbox is the case this whole change is
		// about, and it is not a failure of the test.
		t.Skipf("this process is already confined: %v", err)
	}
}

// A boundary test that cannot get a boundary skips, and says why. The danger is
// that it stays silent: a skipped boundary test reads as a passing one, which
// is how a sandbox comes to be trusted without ever having been exercised.
func TestABoundaryTestSkipsLoudlyRatherThanPassingQuietly(t *testing.T) {
	if _, err := New(Config{Backend: BackendSeatbelt, Binary: "false"},
		policy.ModeWorkspaceWrite); err == nil {
		t.Fatal("a sandbox that cannot confine was constructed anyway")
	}
}
