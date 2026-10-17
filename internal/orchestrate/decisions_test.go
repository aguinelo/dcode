package orchestrate

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

const testProjectID = "0123456789abcdef01234567"

func testProject(t *testing.T) protocol.Project {
	t.Helper()
	return protocol.Project{ID: testProjectID, Root: t.TempDir()}
}

func openLog(t *testing.T, dir string, p protocol.Project) *DecisionLog {
	t.Helper()
	l, err := OpenDecisionLog(dir, p)
	if err != nil {
		t.Fatalf("OpenDecisionLog: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}

func logPath(dir string) string {
	return filepath.Join(dir, testProjectID, "decisions.jsonl")
}

// readLines decodes every line of the log, failing on one that does not parse.
func readLines(t *testing.T, dir string) []Decision {
	t.Helper()
	data, err := os.ReadFile(logPath(dir))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	var out []Decision
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, maxLine), maxLine+1)
	for n := 1; sc.Scan(); n++ {
		var d Decision
		if err := json.Unmarshal(sc.Bytes(), &d); err != nil {
			t.Fatalf("line %d does not parse: %v\n%s", n, err, sc.Bytes())
		}
		out = append(out, d)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// assertEmpty fails unless the log exists and holds nothing.
func assertEmpty(t *testing.T, dir string) {
	t.Helper()
	fi, err := os.Stat(logPath(dir))
	if err != nil {
		t.Fatalf("the log was not created on open: %v", err)
	}
	if fi.Size() != 0 {
		t.Fatalf("refused decisions reached the file: %d bytes", fi.Size())
	}
}

func dispatch(why string) Decision {
	return Decision{Kind: "dispatch", Child: "c1", Session: "s1", Why: why}
}

// Spec: orchestration .i, step 4 — a decision without why is refused, and
// nothing is written.
func TestADecisionWithoutAReasonIsRefusedAndNothingIsWritten(t *testing.T) {
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))

	for _, why := range []string{"", "   ", "\n\t"} {
		seq, err := l.Record(dispatch(why))
		if !errors.Is(err, errNoReason) {
			t.Errorf("why %q: err = %v, want errNoReason", why, err)
		}
		if seq != 0 {
			t.Errorf("why %q: a refused decision returned seq %d", why, seq)
		}
	}
	assertEmpty(t, dir)

	// A refusal spends no seq: the first decision recorded is still 1.
	seq, err := l.Record(dispatch("three themes, three children"))
	if err != nil || seq != 1 {
		t.Fatalf("first recorded decision: seq %d, err %v; want 1", seq, err)
	}
}

func TestAnUnknownKindIsRefusedAndNothingIsWritten(t *testing.T) {
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))

	for _, kind := range []string{"", "dispatched", "gate.passed", "Dispatch"} {
		if _, err := l.Record(Decision{Kind: kind, Why: "because"}); !errors.Is(err, errUnknownKind) {
			t.Errorf("kind %q: err = %v, want errUnknownKind", kind, err)
		}
	}
	assertEmpty(t, dir)
}

func TestADecisionForAnotherProjectIsRefused(t *testing.T) {
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))

	d := dispatch("because")
	d.Project = "ffffffffffffffffffffffff"
	if _, err := l.Record(d); err == nil {
		t.Fatal("a decision naming another project was recorded in this one's log")
	}
}

// Every decision is one line, with the fields of .p §9.1 filled by the log.
func TestEachDecisionIsOneLineWithItsReason(t *testing.T) {
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))
	at := time.Date(2026, 10, 17, 21, 4, 5, 0, time.FixedZone("BRT", -3*3600))
	l.now = func() time.Time { return at }

	measured := json.RawMessage(`{"command":"make check","exit":2}`)
	seq, err := l.Record(Decision{
		Kind: "gate.measured", Child: "c1", Session: "s1",
		Why: "the gate is the harness's", Claim: "make check passed", Measured: measured,
	})
	if err != nil || seq != 1 {
		t.Fatalf("Record: seq %d, err %v", seq, err)
	}

	data, err := os.ReadFile(logPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"v":1,"seq":1,"at":"2026-10-18T00:04:05Z","project":"` + testProjectID +
		`","kind":"gate.measured","child":"c1","session":"s1","why":"the gate is the harness's",` +
		`"claim":"make check passed","measured":{"command":"make check","exit":2}}` + "\n"
	if string(data) != want {
		t.Errorf("line:\n got %s\nwant %s", data, want)
	}
}

// Spec: orchestration .i, step 4 — reopening the log continues seq without a
// gap; .p §9.1 — seq has no gap nor repetition, across restarts.
func TestReopeningTheLogContinuesTheSeqWithoutAGap(t *testing.T) {
	dir := t.TempDir()
	p := testProject(t)

	l, err := OpenDecisionLog(dir, p)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		if seq, err := l.Record(dispatch(fmt.Sprintf("decision %d", i))); err != nil || seq != uint64(i) {
			t.Fatalf("decision %d: seq %d, err %v", i, seq, err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	again := openLog(t, dir, p)
	seq, err := again.Record(dispatch("after the restart"))
	if err != nil || seq != 4 {
		t.Fatalf("first decision after reopening: seq %d, err %v; want 4", seq, err)
	}

	lines := readLines(t, dir)
	if len(lines) != 4 {
		t.Fatalf("%d lines, want 4", len(lines))
	}
	for i, d := range lines {
		if d.Seq != uint64(i+1) {
			t.Errorf("line %d carries seq %d", i+1, d.Seq)
		}
	}
}

// Spec: orchestration .i, step 4 — a write that fails is an error returned,
// never a line lost in silence.
func TestAWriteThatFailsIsAnErrorAndNeverALineLostInSilence(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through a read-only directory")
	}

	t.Run("a read-only directory refuses to open the log", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		if l, err := OpenDecisionLog(dir, testProject(t)); err == nil {
			_ = l.Close()
			t.Fatal("a log was opened in a directory nothing can be written to")
		}
	})

	t.Run("a log whose file cannot take the line returns the error", func(t *testing.T) {
		dir := t.TempDir()
		l := openLog(t, dir, testProject(t))
		if _, err := l.Record(dispatch("first")); err != nil {
			t.Fatal(err)
		}
		_ = l.f.Close() // the descriptor is gone: the next write fails

		seq, err := l.Record(dispatch("second"))
		if err == nil {
			t.Fatal("a decision that could not be written was reported as recorded")
		}
		if seq != 0 {
			t.Errorf("a failed write returned seq %d", seq)
		}
		if got := readLines(t, dir); len(got) != 1 {
			t.Errorf("%d lines in the log, want the one that was written", len(got))
		}
	})
}

// A crash in the middle of a write leaves a last line with no newline. It was
// never acknowledged, so reopening cuts it — and says what it cut.
func TestATornLastLineIsCutOnReopenAndSaid(t *testing.T) {
	dir := t.TempDir()
	p := testProject(t)

	l, err := OpenDecisionLog(dir, p)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		if _, err := l.Record(dispatch(fmt.Sprintf("decision %d", i))); err != nil {
			t.Fatal(err)
		}
	}
	_ = l.Close()

	torn := `{"v":1,"seq":3,"at":"2026-10`
	f, err := os.OpenFile(logPath(dir), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(torn); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	again := openLog(t, dir, p)
	if got := string(again.Torn()); got != torn {
		t.Errorf("Torn() = %q, want the bytes that were cut, %q", got, torn)
	}
	seq, err := again.Record(dispatch("after the crash"))
	if err != nil || seq != 3 {
		t.Fatalf("first decision after the torn line: seq %d, err %v; want 3", seq, err)
	}
	if got := readLines(t, dir); len(got) != 3 {
		t.Errorf("%d readable lines, want 3", len(got))
	}
}

func TestALogWithNoTornLineSaysNothingWasCut(t *testing.T) {
	l := openLog(t, t.TempDir(), testProject(t))
	if got := l.Torn(); len(got) != 0 {
		t.Errorf("Torn() = %q on a fresh log", got)
	}
}

// A complete line that does not read, or a seq that skips or repeats, is a log
// this writer did not leave. Extending it would carry the gap forward, so the
// log refuses to open and names the line.
func TestALogThatBreaksItsSeqRefusesToOpen(t *testing.T) {
	line := func(seq int) string {
		return fmt.Sprintf(`{"v":1,"seq":%d,"at":"2026-10-17T00:00:00Z","project":"%s","kind":"dispatch","why":"x"}`+"\n",
			seq, testProjectID)
	}
	for name, body := range map[string]string{
		"an unreadable line":   line(1) + "not json\n" + line(2),
		"a gap":                line(1) + line(3),
		"a repetition":         line(1) + line(1),
		"a first seq above 1":  line(2),
		"an unknown version":   strings.Replace(line(1), `"v":1`, `"v":2`, 1),
		"an empty line inside": line(1) + "\n" + line(2),
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, testProjectID), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(logPath(dir), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			l, err := OpenDecisionLog(dir, testProject(t))
			if err == nil {
				_ = l.Close()
				t.Fatal("the log opened")
			}
			if !strings.Contains(err.Error(), ":2:") && !strings.Contains(err.Error(), ":1:") {
				t.Errorf("the error does not name the line: %v", err)
			}
		})
	}
}

// Spec: orchestration .i, step 4 — a line over 64 KiB cuts its largest field
// and says how much.
func TestALineOver64KiBCutsTheLargestFieldAndSaysHowMuch(t *testing.T) {
	cases := map[string]func(*Decision, string){
		"claim": func(d *Decision, s string) { d.Claim = s },
		"why":   func(d *Decision, s string) { d.Why = s },
		"data": func(d *Decision, s string) {
			raw, _ := json.Marshal(map[string]string{"output": s})
			d.Data = raw
		},
	}
	for name, set := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			l := openLog(t, dir, testProject(t))

			big := strings.Repeat("ação <&> ", 20000) // ~200 KiB, multibyte and escaped runes
			d := Decision{Kind: "child.completed", Child: "c1", Why: "the child ended", Claim: "short"}
			set(&d, big)
			if _, err := l.Record(d); err != nil {
				t.Fatalf("Record: %v", err)
			}

			data, err := os.ReadFile(logPath(dir))
			if err != nil {
				t.Fatal(err)
			}
			if len(data) > maxLine {
				t.Fatalf("the line is %d bytes, over the %d of the ceiling", len(data), maxLine)
			}
			got := readLines(t, dir)
			if len(got) != 1 {
				t.Fatalf("%d lines, want 1", len(got))
			}
			var field string
			switch name {
			case "claim":
				field = got[0].Claim
				if got[0].Why != "the child ended" {
					t.Errorf("a smaller field was cut too: why = %q", got[0].Why)
				}
			case "why":
				field = got[0].Why
				if got[0].Claim != "short" {
					t.Errorf("a smaller field was cut too: claim = %q", got[0].Claim)
				}
			case "data":
				if err := json.Unmarshal(got[0].Data, &field); err != nil {
					t.Fatalf("a cut data field is the text of what was cut: %v (%s)", err, got[0].Data)
				}
				raw, _ := json.Marshal(map[string]string{"output": big})
				big = string(raw)
			}
			kept, note, ok := strings.Cut(field, "…[")
			if !ok {
				t.Fatalf("the cut field does not say it was cut: …%q", tail(field))
			}
			want := fmt.Sprintf("%d bytes cortados]", len(big)-len(kept))
			if note != want {
				t.Errorf("the note says %q, want %q", note, want)
			}
			if !strings.HasPrefix(big, kept) {
				t.Error("what was kept is not the start of the field")
			}
		})
	}
}

func tail(s string) string {
	if len(s) > 40 {
		return s[len(s)-40:]
	}
	return s
}

// Record is called by every step of the pipeline; two at once still give two
// different seq, contiguous, and two whole lines.
func TestConcurrentDecisionsGetDistinctContiguousSeq(t *testing.T) {
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))

	const n = 50
	var wg sync.WaitGroup
	seen := make(chan uint64, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			seq, err := l.Record(dispatch(fmt.Sprintf("decision %d", i)))
			if err != nil {
				t.Error(err)
			}
			seen <- seq
		}()
	}
	wg.Wait()
	close(seen)

	got := map[uint64]bool{}
	for s := range seen {
		got[s] = true
	}
	for s := uint64(1); s <= n; s++ {
		if !got[s] {
			t.Errorf("seq %d was never returned", s)
		}
	}
	lines := readLines(t, dir)
	for i, d := range lines {
		if d.Seq != uint64(i+1) {
			t.Errorf("line %d carries seq %d", i+1, d.Seq)
		}
	}
}

func TestAClosedLogRefusesADecision(t *testing.T) {
	l, err := OpenDecisionLog(t.TempDir(), testProject(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Record(dispatch("too late")); err == nil {
		t.Fatal("a closed log recorded a decision")
	}
}

func TestTheLogIsOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	openLog(t, dir, testProject(t))

	for path, want := range map[string]os.FileMode{
		filepath.Join(dir, testProjectID): 0o700,
		logPath(dir):                      0o600,
	} {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := fi.Mode().Perm(); got != want {
			t.Errorf("%s is %o, want %o", filepath.Base(path), got, want)
		}
	}
}

func TestOpeningRefusesWhatCannotBeAProjectsLog(t *testing.T) {
	root := t.TempDir()
	for name, c := range map[string]struct {
		dir string
		p   protocol.Project
	}{
		"a relative directory":    {"orchestrate", protocol.Project{ID: testProjectID, Root: root}},
		"an id that is not hex":   {t.TempDir(), protocol.Project{ID: "../../../../etc/passwd000", Root: root}},
		"an id of the wrong size": {t.TempDir(), protocol.Project{ID: "abc", Root: root}},
		"the project's own tree":  {filepath.Join(root, ".dcode", "orchestrate"), protocol.Project{ID: testProjectID, Root: root}},
		"the project's root":      {root, protocol.Project{ID: testProjectID, Root: root}},
	} {
		t.Run(name, func(t *testing.T) {
			if l, err := OpenDecisionLog(c.dir, c.p); err == nil {
				_ = l.Close()
				t.Fatal("the log opened")
			}
		})
	}
}
