package tools

import (
	"strings"
	"testing"
)

func TestEchoDiffReturnsTheDiffOnlyWhereTheModelCannotDeriveIt(t *testing.T) {
	cases := []struct {
		name       string
		mode       string
		replaceAll bool
		count      int
		want       bool
	}{
		// The one case the whole rule exists for: the model does not know
		// where the N sites were.
		{"multi, replace_all over 3", EchoDiffMulti, true, 3, true},

		// old_string is unique, so the site is determined and the model can
		// work out what the file now says.
		{"multi, single occurrence", EchoDiffMulti, false, 1, false},
		{"multi, replace_all over exactly 1", EchoDiffMulti, true, 1, false},

		{"never, replace_all over 3", EchoDiffNever, true, 3, false},
		{"never, single", EchoDiffNever, false, 1, false},

		{"always, single", EchoDiffAlways, false, 1, true},
		{"always, replace_all over 3", EchoDiffAlways, true, 3, true},

		// An unset or misspelled mode falls to the default rather than to
		// silence: losing the diff is the failure that is invisible.
		{"unset behaves as multi", "", true, 3, true},
		{"unset, single", "", false, 1, false},
		{"nonsense behaves as multi", "sometimes", true, 3, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := echoDiff(c.mode, c.replaceAll, c.count); got != c.want {
				t.Errorf("echoDiff(%q, %v, %d) = %v, want %v", c.mode, c.replaceAll, c.count, got, c.want)
			}
		})
	}
}

// The behaviour end to end, through the real Edit tool.
func TestEditEchoesTheDiffToTheModelOnlyOnAMultiOccurrenceReplaceAll(t *testing.T) {
	const body = "alpha\nbeta\nalpha\ngamma\nalpha\n"

	cases := []struct {
		name       string
		mode       string
		replaceAll bool
		old, new   string
		wantEcho   bool
	}{
		{"multi mode, three occurrences", EchoDiffMulti, true, "alpha", "delta", true},
		{"multi mode, unique occurrence", EchoDiffMulti, false, "beta", "delta", false},
		{"never mode, three occurrences", EchoDiffNever, true, "alpha", "delta", false},
		{"always mode, unique occurrence", EchoDiffAlways, false, "beta", "delta", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, ws := setup(t)
			s.Limits.EditEchoDiff = c.mode
			p := writeFileT(t, ws, "f.txt", body)
			s.MarkRead(p, body, 0)

			res := run(t, Edit{}, s, EditInput{
				Path: "f.txt", OldString: c.old, NewString: c.new, ReplaceAll: c.replaceAll,
			})
			if res.Meta.Diff == "" {
				t.Fatal("the client lost its diff; Meta.Diff must be filled in every mode")
			}

			echoed := strings.Contains(res.Output, "@@") || strings.Contains(res.Output, "--- ")
			if echoed != c.wantEcho {
				t.Errorf("diff echoed to the model = %v, want %v.\nOutput:\n%s",
					echoed, c.wantEcho, res.Output)
			}
			if !strings.Contains(res.Output, "replacement(s)") {
				t.Error("the count line is gone; the diff supplements it, never replaces it")
			}
		})
	}
}

// write dictated every byte, so there is nothing the model could learn from
// the diff. No mode changes that — the key has no say over write at all.
func TestWriteNeverEchoesADiffInAnyMode(t *testing.T) {
	for _, mode := range []string{EchoDiffNever, EchoDiffMulti, EchoDiffAlways} {
		t.Run(mode, func(t *testing.T) {
			s, ws := setup(t)
			s.Limits.EditEchoDiff = mode
			const before = "one\ntwo\nthree\n"
			p := writeFileT(t, ws, "f.txt", before)
			s.MarkRead(p, before, 0)

			res := run(t, Write{}, s, WriteInput{Path: "f.txt", Content: "four\nfive\n"})
			if strings.Contains(res.Output, "@@") {
				t.Errorf("write echoed a diff in mode %q:\n%s", mode, res.Output)
			}
		})
	}
}

// An echoed diff is capped and says so, like every other bounded output (RN-5).
// A diff truncated in silence is the worst case: the model concludes about the
// part it did not see, with the confidence of having seen all of it.
func TestAnEchoedDiffThatIsTruncatedSaysSo(t *testing.T) {
	var b strings.Builder
	for i := 0; i < DiffMaxLines+200; i++ {
		b.WriteString("target\n")
	}
	body := b.String()

	s, ws := setup(t)
	s.Limits.EditEchoDiff = EchoDiffMulti
	p := writeFileT(t, ws, "big.txt", body)
	s.MarkRead(p, body, 0)

	res := run(t, Edit{}, s, EditInput{
		Path: "big.txt", OldString: "target", NewString: "changed", ReplaceAll: true,
	})
	if !strings.Contains(res.Output, "truncated") {
		t.Errorf("an echoed diff was cut without saying so:\n%s", res.Output)
	}
}
