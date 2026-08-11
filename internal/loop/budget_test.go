package loop

import (
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// engineAt builds an engine whose session already occupies a known share of a
// small window, so the band arithmetic can be driven without a provider.
func engineAt(t *testing.T, chars int) *Engine {
	t.Helper()
	cfg := ce.DefaultConfig()
	cfg.Window = 1000
	e := New(Config{CtxConfig: cfg, BudgetNotice: true}, ce.Session{Instructions: "x"})
	fill(t, e, chars)
	return e
}

func fill(t *testing.T, e *Engine, chars int) {
	t.Helper()
	body := make([]byte, chars)
	for i := range body {
		body[i] = 'a'
	}
	e.session.History = []ce.Message{{Role: ce.RoleUser, Text: string(body)}}
}

func TestNoBudgetReminderOnAShortSession(t *testing.T) {
	e := engineAt(t, 100)
	if got := e.crossBudget(); got != ce.BandNone {
		t.Fatalf("a short session announced %v", got)
	}
}

func TestABandIsAnnouncedOnceAndNotAgain(t *testing.T) {
	e := engineAt(t, 1800) // ~0.57 of a 1000-token window after the margin

	first := e.crossBudget()
	if first == ce.BandNone {
		t.Fatalf("nothing announced at fraction %v", ce.Fraction(e.session, e.cfg.CtxConfig))
	}
	if again := e.crossBudget(); again != ce.BandNone {
		t.Errorf("the same band was announced twice: %v", again)
	}
}

func TestCrossingHigherAnnouncesAgain(t *testing.T) {
	e := engineAt(t, 1800)
	low := e.crossBudget()
	if low == ce.BandNone {
		t.Fatal("nothing announced at the first level")
	}
	fill(t, e, 2600)
	high := e.crossBudget()
	if high <= low {
		t.Fatalf("growing the context announced %v, want something above %v", high, low)
	}
}

// Compaction drops the fraction, so the climb back up is genuinely new
// information and is announced again.
func TestTheWarningRearmsAfterCompaction(t *testing.T) {
	e := engineAt(t, 2600)
	if e.crossBudget() == ce.BandNone {
		t.Fatal("nothing announced at the high level")
	}
	fill(t, e, 100) // as if history had been summarised away
	if got := e.crossBudget(); got != ce.BandNone {
		t.Errorf("announced %v on the way down", got)
	}
	fill(t, e, 2600)
	if e.crossBudget() == ce.BandNone {
		t.Fatal("the climb back was swallowed; the warning never rearmed")
	}
}

func TestBudgetNoticeOffAnnouncesNothing(t *testing.T) {
	e := engineAt(t, 2600)
	e.cfg.BudgetNotice = false
	if got := e.crossBudget(); got != ce.BandNone {
		t.Fatalf("announced %v with the notice switched off", got)
	}
}
