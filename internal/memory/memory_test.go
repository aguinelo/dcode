package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func workspace(t *testing.T, body string) string {
	t.Helper()
	ws := t.TempDir()
	if body == "" {
		return ws
	}
	if err := os.MkdirAll(filepath.Join(ws, ".dcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(ws), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return ws
}

// A workspace that never learned anything is every workspace on its first day.
// Nothing new may fail because there is nothing to read.
func TestAWorkspaceWithNoMemoryReadsAsEmpty(t *testing.T) {
	got, err := Read(workspace(t, ""))
	if err != nil {
		t.Fatalf("a missing memory was an error: %v", err)
	}
	if len(got.Entries) != 0 || len(got.Malformed) != 0 {
		t.Errorf("got %+v, want nothing", got)
	}
}

// The whole grammar: a typed header, provenance in a comment, a body.
func TestAMemoryIsAKindASubjectAndABody(t *testing.T) {
	ws := workspace(t, `## gotcha: make test precisa de go generate antes
<!-- learned 2026-08-18 · commit a2c6e69 -->

`+"`make test` falha quando os arquivos gerados estão velhos."+`
Rode `+"`go generate ./...`"+` antes.

## decision: -race fica na CI
<!-- learned 2026-08-01 · commit deadbee -->

Custa dois minutos e paga por si na primeira corrida de dados.
`)

	got, err := Read(ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("read %d memories, want 2: %+v", len(got.Entries), got.Entries)
	}

	first := got.Entries[0]
	if first.Kind != KindGotcha {
		t.Errorf("kind = %q", first.Kind)
	}
	if first.Subject != "make test precisa de go generate antes" {
		t.Errorf("subject = %q", first.Subject)
	}
	if first.Learned != "2026-08-18" || first.Commit != "a2c6e69" {
		t.Errorf("provenance = %q / %q", first.Learned, first.Commit)
	}
	if !strings.Contains(first.Body, "go generate") {
		t.Errorf("body = %q", first.Body)
	}
	// The body stops at the next memory rather than swallowing it.
	if strings.Contains(first.Body, "-race") {
		t.Errorf("the first memory swallowed the second: %q", first.Body)
	}
	if got.Entries[1].Kind != KindDecision {
		t.Errorf("second kind = %q", got.Entries[1].Kind)
	}
}

// A block that looked like a memory and was not is reported, not swallowed.
//
// A session must not fail over a crooked block — that would be the memory
// holding the product hostage. But dropping it in silence is knowledge lost with
// nobody told, which is worse than the crooked block.
func TestACrookedBlockIsReportedAndTheRestSurvives(t *testing.T) {
	ws := workspace(t, `## gotcha: this one is fine

body here.

## nonsense: not a kind

## gotcha:

## decision: this one is fine too

body.
`)

	got, err := Read(ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("read %d memories, want the two that parse: %+v", len(got.Entries), got.Entries)
	}
	if len(got.Malformed) != 2 {
		t.Fatalf("reported %v, want both crooked blocks", got.Malformed)
	}
	for _, m := range got.Malformed {
		if !strings.Contains(m, "nonsense") && !strings.Contains(m, "gotcha:") {
			t.Errorf("unexpected report: %q", m)
		}
	}
}

// Somebody wrote it by hand and did not add a comment nobody told them about.
// A memory with no provenance is still a memory.
func TestAMemoryWrittenByHandNeedsNoProvenance(t *testing.T) {
	ws := workspace(t, "## convention: tabelas de teste, nunca t.Run solto\n\nporque sim.\n")
	got, err := Read(ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("got %+v", got.Entries)
	}
	if got.Entries[0].Learned != "" || got.Entries[0].Commit != "" {
		t.Errorf("invented provenance: %+v", got.Entries[0])
	}
	if got.Entries[0].Kind != KindConvention {
		t.Errorf("kind = %q", got.Entries[0].Kind)
	}
}

// The kind is matched without case, because a person typing it should not have
// to remember which.
func TestTheKindIsMatchedWithoutCase(t *testing.T) {
	ws := workspace(t, "## Gotcha: shouting\n\nbody.\n")
	got, _ := Read(ws)
	if len(got.Entries) != 1 || got.Entries[0].Kind != KindGotcha {
		t.Errorf("got %+v", got.Entries)
	}
}

// The list of kinds is closed, and Valid is what closes it.
func TestOnlyThreeKindsAreValid(t *testing.T) {
	for _, k := range Kinds() {
		if !k.Valid() {
			t.Errorf("%q is in the list and not valid", k)
		}
	}
	for _, k := range []Kind{"", "note", "todo", "GOTCHA "} {
		if Kind(k).Valid() {
			t.Errorf("%q was accepted", k)
		}
	}
	if len(Kinds()) != 3 {
		t.Errorf("there are %d kinds; the list is meant to be closed at three", len(Kinds()))
	}
}

// A memory holding a pasted stack trace must survive the round trip. The
// scanner's default limit would cut it and report the tail as its own line.
func TestALongMemorySurvives(t *testing.T) {
	long := strings.Repeat("x", 200_000)
	ws := workspace(t, "## gotcha: a long one\n\n"+long+"\n")
	got, err := Read(ws)
	if err != nil {
		t.Fatalf("a long memory could not be read: %v", err)
	}
	if len(got.Entries) != 1 || len(got.Entries[0].Body) < 200_000 {
		t.Fatalf("the body was cut: %d memories, %d bytes",
			len(got.Entries), len(got.Entries[0].Body))
	}
}

// A file that cannot be read at all is an error, distinct from a file that is
// not there. One is a workspace with no memory; the other is a memory nobody
// can reach, and reporting them the same way hides the second.
func TestAMemoryThatCannotBeReadIsAnError(t *testing.T) {
	ws := t.TempDir()
	// A directory where the file should be.
	if err := os.MkdirAll(Path(ws), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(ws); err == nil {
		t.Fatal("an unreadable memory reported success")
	}
}
