package contextengine

// Fraction is how much of the window the assembled context occupies, in [0,1].
//
// The number already existed and was already computed on every iteration of the
// loop — Plan compares exactly this against the compaction trigger. What it was
// not, was reachable by anyone who could act on it. This exposes it.
//
// Pure, like Estimate, and the same arithmetic Plan performs.
func Fraction(s Session, cfg Config) float64 {
	cfg = withDefaults(cfg)
	if cfg.Window <= 0 {
		return 0
	}
	msgs, err := Assemble(s)
	if err != nil {
		return 0
	}
	return float64(Estimate(msgs, cfg)) / float64(cfg.Window)
}

// Band is how full the context was the last time the model was told.
//
// It lives in session state rather than being derived per turn because the
// announcement is edge-triggered: emitting while the fraction is merely ABOVE a
// threshold repeats the same reminder every turn, which costs tokens and, worse,
// produces habituation. A warning that is always there stops being read.
type Band int

const (
	BandNone Band = iota // nothing announced yet
	Band60
	Band80
	Band92
)

// DefaultBands are the announcement thresholds, ascending, as fractions of the
// BUDGET rather than of the window.
//
// The distinction is forced, and it is the one correction this change makes to
// its own spec. The spec declares the bands as fractions of the window — 0.60,
// 0.80, 0.92 — and separately requires every band to sit below CompactAt, which
// defaults to 0.80. Both cannot hold: two of the three bands are unreachable,
// because the context is cut at 0.80 and never gets to 0.92.
//
// Read as fractions of the budget, both statements hold and the meaning
// improves. The budget is the space before compaction, so "you are at 92%" says
// 92% of what you get before your memory is cut — which is exactly the thing
// the model can act on. Against the window it would have been a number about a
// limit that never arrives.
var DefaultBands = []float64{0.60, 0.80, 0.92}

// bandEpsilon absorbs binary floating-point error at a threshold.
const bandEpsilon = 1e-9

// BandFor returns the band a window fraction falls in. Pure.
//
// compactAt is a parameter rather than a package constant because the trigger is
// configuration: a session that compacts at 0.5 has a smaller budget, and the
// bands have to move with it or they stop meaning anything.
func BandFor(f, compactAt float64) Band {
	if compactAt <= 0 {
		return BandNone
	}
	// The epsilon is not defensive noise. Fraction comes from a character
	// heuristic with a safety margin on top, so a difference of 1e-16 carries
	// no information — while without it, a value sitting exactly on a
	// threshold falls to the band below, because 0.64/0.80 is 0.7999…
	// in binary floating point. Surprising, and about nothing.
	spent := f/compactAt + bandEpsilon // fraction of the budget, not of the window
	switch {
	case spent >= DefaultBands[2]:
		return Band92
	case spent >= DefaultBands[1]:
		return Band80
	case spent >= DefaultBands[0]:
		return Band60
	default:
		return BandNone
	}
}

// Crossed reports the band to announce, and whether to announce at all.
//
// Upward only. Going back down — which is what compaction does — announces
// nothing, but it does rearm: the returned band becomes the new high-water
// mark, so climbing back up is news again, and genuinely is.
func Crossed(announced Band, f, compactAt float64) (Band, bool) {
	now := BandFor(f, compactAt)
	if now > announced {
		return now, true
	}
	return now, false
}
