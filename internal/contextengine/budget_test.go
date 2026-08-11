package contextengine

import (
	"strings"
	"testing"
)

func TestBandForMapsWindowFractionsToBands(t *testing.T) {
	// Budget is 0.80 of the window, so the bands land at 0.48, 0.64 and 0.736
	// of the window.
	const compactAt = 0.80
	cases := []struct {
		f    float64
		want Band
	}{
		{0, BandNone},
		{0.47, BandNone},
		{0.48, Band60}, // exactly on the boundary counts as crossed
		{0.63, Band60},
		{0.64, Band80},
		{0.73, Band80},
		{0.736, Band92},
		{0.79, Band92}, // still the top band right up to the cut
	}
	for _, c := range cases {
		if got := BandFor(c.f, compactAt); got != c.want {
			t.Errorf("BandFor(%v, %v) = %v, want %v", c.f, compactAt, got, c.want)
		}
	}
}

// Every threshold must land below the compaction trigger. A warning that
// arrives together with the cut arrives too late to act on, which is the defect
// this whole change exists to remove.
func TestEveryBandLandsBelowTheCompactionTrigger(t *testing.T) {
	for _, compactAt := range []float64{0.5, 0.8, 0.95} {
		for _, b := range DefaultBands {
			if got := b * compactAt; got >= compactAt {
				t.Errorf("band %v at CompactAt %v lands at %v, which is not below the cut", b, compactAt, got)
			}
		}
	}
}

// The bands move with the trigger. A session that compacts at 0.5 has a smaller
// budget, and a band fixed to the window would fire after the cut.
func TestBandsFollowTheConfiguredTrigger(t *testing.T) {
	if got := BandFor(0.48, 0.50); got != Band92 {
		t.Errorf("at CompactAt 0.5, 0.48 of the window is 96%% of the budget; got %v", got)
	}
	if got := BandFor(0.48, 0.80); got != Band60 {
		t.Errorf("at CompactAt 0.8, 0.48 of the window is 60%% of the budget; got %v", got)
	}
}

func TestAnUnsetTriggerAnnouncesNothing(t *testing.T) {
	if got := BandFor(0.99, 0); got != BandNone {
		t.Fatalf("BandFor with no trigger = %v, want BandNone rather than a divide by zero", got)
	}
}

func TestAnnouncementIsEdgeTriggeredNotLevelTriggered(t *testing.T) {
	// First crossing announces.
	band, announce := Crossed(BandNone, 0.52, 0.80)
	if !announce || band != Band60 {
		t.Fatalf("first crossing into 60 gave (%v, %v)", band, announce)
	}
	// Staying inside the same band announces nothing. Repeating it every turn
	// costs tokens and produces habituation: a warning always on screen stops
	// being read.
	if _, announce := Crossed(Band60, 0.56, 0.80); announce {
		t.Error("announced again without crossing anything")
	}
	if _, announce := Crossed(Band60, 0.63, 0.80); announce {
		t.Error("announced again at the top of the same band")
	}
	// The next band up is news.
	band, announce = Crossed(Band60, 0.65, 0.80)
	if !announce || band != Band80 {
		t.Fatalf("crossing into 80 gave (%v, %v)", band, announce)
	}
}

func TestFallingBackDownAnnouncesNothingButRearms(t *testing.T) {
	// Compaction drops the fraction. Nothing is announced on the way down.
	band, announce := Crossed(Band80, 0.10, 0.80)
	if announce {
		t.Error("announced on the way down")
	}
	if band != BandNone {
		t.Fatalf("band after the drop = %v, want BandNone so the climb is news again", band)
	}
	// Climbing back is genuinely new information, and worth as much as the
	// first time.
	if _, announce := Crossed(band, 0.52, 0.80); !announce {
		t.Error("the second climb into 60 was swallowed; the reminder never rearmed")
	}
}

func TestFractionIsTheSameArithmeticPlanUses(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 1000

	s := Session{Instructions: "x", History: []Message{
		{Role: RoleUser, Text: strings.Repeat("a ", 400)},
	}}
	msgs, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}
	want := float64(Estimate(msgs, cfg)) / float64(cfg.Window)
	if got := Fraction(s, cfg); got != want {
		t.Fatalf("Fraction = %v, want %v — it must be the number Plan compares", got, want)
	}
}

func TestFractionIsPureAndDoesNotDriftBetweenCalls(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 1000
	s := Session{Instructions: "x", History: []Message{{Role: RoleUser, Text: "hello"}}}

	first := Fraction(s, cfg)
	for i := 0; i < 5; i++ {
		if got := Fraction(s, cfg); got != first {
			t.Fatalf("Fraction drifted on call %d: %v != %v", i, got, first)
		}
	}
}

func TestAWindowlessSessionReportsZeroRatherThanDividingByZero(t *testing.T) {
	if got := Fraction(Session{Instructions: "x"}, Config{}); got != 0 {
		t.Fatalf("Fraction with no window = %v, want 0", got)
	}
}

// A short session must produce no budget reminder at all. This is the
// deterministic half of no-budget-noise-when-low.
func TestAShortSessionCrossesNothing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 100000
	s := Session{Instructions: "x", History: []Message{{Role: RoleUser, Text: "hi"}}}

	if _, announce := Crossed(BandNone, Fraction(s, cfg), cfg.CompactAt); announce {
		t.Fatal("a two-word session announced a budget band")
	}
}
