package loop

import (
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// The clock is injectable so a test can assert a duration without waiting for
// one. Nothing configured means the real clock, and a session that fell back to
// a frozen zero time would date every turn to 1970.
func TestTheEngineFallsBackToTheRealClock(t *testing.T) {
	e := New(Config{}, ce.Session{Instructions: "x"})
	before := time.Now()
	got := e.now()
	if got.Before(before.Add(-time.Second)) || got.After(time.Now().Add(time.Second)) {
		t.Errorf("now() = %v, want the real clock", got)
	}

	frozen := time.Unix(1_000_000, 0)
	e = New(Config{Now: func() time.Time { return frozen }}, ce.Session{Instructions: "x"})
	if !e.now().Equal(frozen) {
		t.Errorf("now() = %v, want the injected clock", e.now())
	}
}

// The stall limit has a default because a session with none configured must
// still stop going round. Zero would mean the loop gives up before it has tried
// anything, and a missing setting is not a request to do that.
func TestTheStallLimitHasADefaultThatIsNotZero(t *testing.T) {
	e := New(Config{}, ce.Session{Instructions: "x"})
	if got := e.stallLimit(); got != 2 {
		t.Errorf("stallLimit = %d, want the default of 2", got)
	}
	e = New(Config{MaxStallCycles: 5}, ce.Session{Instructions: "x"})
	if got := e.stallLimit(); got != 5 {
		t.Errorf("stallLimit = %d, want the configured 5", got)
	}
}

// The budget notice is off unless asked for, and off means silent — not "band
// none computed from a window nobody set". A notice that fires with no window
// would announce a crossing of a limit that does not exist.
func TestTheBudgetNoticeIsSilentUntilItIsAskedFor(t *testing.T) {
	e := New(Config{}, ce.Session{Instructions: "x"})
	if got := e.crossBudget(); got != ce.BandNone {
		t.Errorf("band = %v with the notice off, want none", got)
	}
}
