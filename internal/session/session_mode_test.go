package session

import (
	"sync"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// TestModeToSandbox pins the mapping a session advertises. Drift here is
// silent: the engine would run under a mode the user did not pick, and the
// footer would show "auto" while the boundary was still there.
func TestModeToSandbox(t *testing.T) {
	cases := []struct {
		name  string
		wantM policy.SandboxMode
		wantP policy.ApprovalPolicy
	}{
		{protocol.ModePlan, policy.ModeReadOnly, policy.PolicyNever},
		{protocol.ModeAssist, policy.ModeWorkspaceWrite, policy.PolicyOnRequest},
		{protocol.ModeAuto, policy.ModeFullAccess, policy.PolicyOnRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotM, gotP := modeToSandbox(c.name)
			if gotM != c.wantM {
				t.Errorf("mode = %q, want %q", gotM, c.wantM)
			}
			if gotP != c.wantP {
				t.Errorf("policy = %q, want %q", gotP, c.wantP)
			}
		})
	}
}

// TestModeFromIsTheExactInverse walks every mode out and back.
//
// The two directions are written separately, so nothing but a test stops them
// from disagreeing — and a disagreement is a session labelled as a mode it is
// not running under.
func TestModeFromIsTheExactInverse(t *testing.T) {
	for _, name := range []string{protocol.ModePlan, protocol.ModeAssist, protocol.ModeAuto} {
		if got := modeFrom(modeToSandbox(name)); got != name {
			t.Errorf("modeFrom(modeToSandbox(%q)) = %q, want %q", name, got, name)
		}
	}
}

// TestAPairThatIsNoModeHasNoName is the honest-silence case.
//
// read-only that still asks is a legitimate configuration and it is none of
// the three. Naming it anyway would put a word on the status bar that the
// engine does not answer to; the empty string is what modeSegment already
// knows not to draw.
func TestAPairThatIsNoModeHasNoName(t *testing.T) {
	if got := modeFrom(policy.ModeReadOnly, policy.PolicyOnRequest); got != "" {
		t.Errorf("read-only + on-request named %q, want no name", got)
	}
	if got := modeFrom(policy.ModeFullAccess, policy.PolicyNever); got != "" {
		t.Errorf("full-access + never named %q, want no name", got)
	}
}

func modeSession(t *testing.T, m policy.SandboxMode, p policy.ApprovalPolicy) *Session {
	t.Helper()
	eng := loop.New(loop.Config{Mode: m, Policy: p}, ce.Session{})
	return New("s1", "/w", "model", string(m), eng, NewEventLog("s1", 100, time.Now), time.Now)
}

// TestASessionSaysTheModeItIsActuallyIn is the defect this file was written for.
//
// The session used to be born labelled "assist" whatever the engine was
// running, so a full-access session showed the quiet badge of the bounded one
// — and the badge is the whole point of having modes.
func TestASessionSaysTheModeItIsActuallyIn(t *testing.T) {
	cases := []struct {
		mode   policy.SandboxMode
		policy policy.ApprovalPolicy
		want   string
	}{
		{policy.ModeReadOnly, policy.PolicyNever, protocol.ModePlan},
		{policy.ModeWorkspaceWrite, policy.PolicyOnRequest, protocol.ModeAssist},
		{policy.ModeFullAccess, policy.PolicyOnRequest, protocol.ModeAuto},
		{policy.ModeReadOnly, policy.PolicyOnRequest, ""},
	}
	for _, c := range cases {
		t.Run(string(c.mode)+"/"+string(c.policy), func(t *testing.T) {
			got := modeSession(t, c.mode, c.policy).Describe().Mode
			if got != c.want {
				t.Errorf("Describe().Mode = %q, want %q", got, c.want)
			}
		})
	}
}

// TestSwitchingAwayFromAModeYouOnlySeemedToBeInWorks is the consequence.
//
// A session mislabelled "assist" refused to switch TO assist — previous ==
// name, so the no-op fired and the engine was never touched. The command that
// installs the boundary silently did nothing, which is the worst direction for
// this particular no-op to fail in.
func TestSwitchingAwayFromAModeYouOnlySeemedToBeInWorks(t *testing.T) {
	s := modeSession(t, policy.ModeFullAccess, policy.PolicyOnRequest)
	if got := s.Describe().Mode; got != protocol.ModeAuto {
		t.Fatalf("precondition: mode = %q, want auto", got)
	}
	before := s.Log.LastSeq()
	if err := s.SetMode(protocol.ModeAssist); err != nil {
		t.Fatalf("SetMode(assist): %v", err)
	}
	if got := s.Describe().Mode; got != protocol.ModeAssist {
		t.Errorf("after SetMode(assist): mode = %q, want assist", got)
	}
	if got := s.Log.LastSeq() - before; got != 1 {
		t.Errorf("events emitted = %d, want 1 (the announcement)", got)
	}
	if got := s.Describe().SandboxMode; got != string(policy.ModeWorkspaceWrite) {
		t.Errorf("the advertised sandbox must follow the switch, got %q", got)
	}
}

// TestSetModeIsANoOpOnlyWhenItIsAlreadyTheMode keeps the silence narrow.
func TestSetModeIsANoOpOnlyWhenItIsAlreadyTheMode(t *testing.T) {
	s := modeSession(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)
	before := s.Log.LastSeq()
	if err := s.SetMode(protocol.ModeAssist); err != nil {
		t.Fatalf("SetMode(assist): %v", err)
	}
	if got := s.Log.LastSeq(); got != before {
		t.Errorf("switching to the mode already in force emitted %d events, want 0", got-before)
	}
}

// TestAnUnknownModeIsRefusedBeforeTheEngine keeps a typo off the boundary.
func TestAnUnknownModeIsRefusedBeforeTheEngine(t *testing.T) {
	s := modeSession(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)
	if err := s.SetMode("yolo"); err == nil {
		t.Fatal("an unknown mode was accepted")
	}
	if got := s.Describe().Mode; got != protocol.ModeAssist {
		t.Errorf("a refused switch must leave the mode alone, got %q", got)
	}
}

// TestConcurrentSwitchesLeaveOneMode covers the check-and-set.
//
// previous used to be read under the lock, the lock released, the engine
// mutated, and only then the field written — so two switches could both pass
// the no-op check and land in either order. Meaningful under -race.
func TestConcurrentSwitchesLeaveOneMode(t *testing.T) {
	s := modeSession(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)

	var wg sync.WaitGroup
	for _, name := range []string{protocol.ModePlan, protocol.ModeAuto, protocol.ModeAssist} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				_ = s.SetMode(name)
			}
		}()
	}
	wg.Wait()

	d := s.Describe()
	if !protocol.ValidMode(d.Mode) {
		t.Fatalf("mode = %q, want one of the three", d.Mode)
	}
	wantSandbox, _ := modeToSandbox(d.Mode)
	if d.SandboxMode != string(wantSandbox) {
		t.Errorf("mode %q advertises sandbox %q, want %q", d.Mode, d.SandboxMode, wantSandbox)
	}
}

// The session says how many criteria it carries, and zero is an answer.
//
// A session with no definition of done reports done at the end of the first
// turn. Someone who typed /loop expecting one has to be told then, and the
// client cannot say it without being told first.
func TestTheSessionReportsHowManyCriteriaItCarries(t *testing.T) {
	s := criteriaSession(t, 2)
	if got := s.Describe().DoneCriteria; got != 2 {
		t.Errorf("Describe reports %d criteria, want 2", got)
	}
	s = criteriaSession(t, 0)
	if got := s.Describe().DoneCriteria; got != 0 {
		t.Errorf("Describe reports %d criteria for an empty set, want 0", got)
	}
}

func criteriaSession(t *testing.T, n int) *Session {
	t.Helper()
	var set loop.DoneSet
	for i := 0; i < n; i++ {
		set.Criteria = append(set.Criteria, loop.Criterion{Name: "c", Command: "true"})
	}
	eng := loop.New(loop.Config{Done: set}, ce.Session{})
	return New("s1", "/w", "model", "workspace-write", eng, NewEventLog("s1", 100, time.Now), time.Now)
}
