package evals

import (
	"context"
	"strings"
	"testing"
)

// The defect this replaced: every tool answered with the string "ok", so a
// model that globbed and grepped concluded its workspace was empty and refused
// the task. It was right about what it had been told, and it scored zero.
func TestTheWorkspaceAnswersWithItsRealContent(t *testing.T) {
	w := workspaceWith(t, map[string]string{
		"stats.go": "package stats\n\ntype Summary struct{ count int }\n",
	})

	out, isErr := w.Execute(context.Background(), "read", []byte(`{"path":"stats.go"}`))
	if isErr {
		t.Fatalf("reading a file that exists failed: %s", out)
	}
	if !strings.Contains(out, "type Summary") {
		t.Errorf("the read did not return the file:\n%s", out)
	}
	if out == "ok" {
		t.Error("the workspace is still answering with the placeholder")
	}
}

// grep has to find things, or a scenario that starts by searching ends before
// it starts.
func TestTheWorkspaceSearchesItsOwnFiles(t *testing.T) {
	w := workspaceWith(t, map[string]string{
		"stats.go":  "package stats\n\ntype Summary struct{ count int }\n",
		"parse.go":  "package stats\n\nfunc Parse() {}\n",
		"README.md": "a summary of nothing\n",
	})

	out, isErr := w.Execute(context.Background(), "grep", []byte(`{"pattern":"type Summary"}`))
	if isErr {
		t.Fatalf("grep failed: %s", out)
	}
	if !strings.Contains(out, "stats.go") {
		t.Errorf("grep did not find the match:\n%s", out)
	}
}

// The tools are the product's own, so a failure carries the product's own
// message. RN-3 makes that text a behaviour surface, and a scenario measuring
// recovery from a hand-written message measures a product that does not exist.
func TestAFailureCarriesTheProductsOwnMessage(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})

	out, isErr := w.Execute(context.Background(), "read", []byte(`{"path":"nope.go"}`))
	if !isErr {
		t.Fatalf("reading a missing file succeeded: %s", out)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("the failure said nothing at all")
	}
}

// The shell is refused rather than faked. Pretending a command ran and
// returned nothing is what produced "shell responses look empty. Let me try
// again with different approaches" — a model burning its rounds on a lie.
func TestTheShellIsRefusedInWordsRatherThanFaked(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})

	out, isErr := w.Execute(context.Background(), "bash", []byte(`{"command":"cat stats.go"}`))
	if !isErr {
		t.Error("the shell reported success for a command that never ran")
	}
	if !strings.Contains(out, "does not execute") {
		t.Errorf("the refusal does not say what happened:\n%s", out)
	}
}

// A write lands in the workspace, so a later read sees it. A scenario whose
// second half depends on its first half cannot work otherwise.
func TestAWriteIsVisibleToTheNextRead(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})
	ctx := context.Background()

	if out, isErr := w.Execute(ctx, "write", []byte(`{"path":"new.go","content":"package stats\n\nfunc New() {}\n"}`)); isErr {
		t.Fatalf("the write failed: %s", out)
	}
	out, isErr := w.Execute(ctx, "read", []byte(`{"path":"new.go"}`))
	if isErr || !strings.Contains(out, "func New") {
		t.Errorf("the write was not visible to the read: %s", out)
	}
}

// A name the registry does not carry must not answer as though it worked.
func TestAnUnknownToolIsAnError(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})
	out, isErr := w.Execute(context.Background(), "delete_file", []byte(`{"path":"stats.go"}`))
	if !isErr {
		t.Errorf("an unknown tool reported success: %s", out)
	}
}

// The shared workspace is the one every scenario explores, and an empty one
// would put every multi-round scenario back where it started.
func TestTheSharedWorkspaceIsReal(t *testing.T) {
	files, err := loadFiles(WorkspaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"stats.go", "internal/config/toml.go", "internal/version/version.go"} {
		if _, ok := files[want]; !ok {
			t.Errorf("the shared workspace has no %s, and scenarios name it by path", want)
		}
	}
}

// A fixture's own files win, so a scenario that needs a file broken in a
// particular way says so without breaking it for everyone else.
func TestAFixtureOverlaysTheSharedWorkspace(t *testing.T) {
	merged := overlay(
		map[string]string{"stats.go": "shared", "other.go": "kept"},
		map[string]string{"stats.go": "own"},
	)
	if merged["stats.go"] != "own" {
		t.Errorf("the fixture did not win: %q", merged["stats.go"])
	}
	if merged["other.go"] != "kept" {
		t.Error("the overlay dropped a shared file the fixture said nothing about")
	}
}

// Every scenario gets the shared workspace without asking for it.
func TestEveryFixtureCarriesAWorkspace(t *testing.T) {
	for _, c := range Contracts {
		f, err := LoadFixture(FixtureRoot, c.ID)
		if err != nil {
			t.Errorf("%s: %v", c.ID, err)
			continue
		}
		if len(f.Files) == 0 {
			t.Errorf("%s runs against an empty workspace, which the model will report as missing", c.ID)
		}
	}
}

// The harness ran tools by calling Execute directly, which skips the gate the
// loop applies. A tool resolves a path without asking whether it may, so
// `read` on /etc/passwd came back with the file — the harness had handed a
// real model unrestricted read access to the machine it was running on.
//
// It is also the wrong measurement: a scenario about staying inside the
// workspace cannot be measured by a harness with no boundary.
func TestNothingOutsideTheWorkspaceCanBeRead(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})

	for _, path := range []string{"/etc/passwd", "/dev/null", "../../../etc/hosts", "/tmp"} {
		out, isErr := w.Execute(context.Background(), "read", []byte(`{"path":"`+path+`"}`))
		if !isErr {
			t.Errorf("reading %s succeeded and returned %d bytes", path, len(out))
		}
	}
}

// And nothing outside it can be written, which is the half that changes the
// machine rather than only reading it.
func TestNothingOutsideTheWorkspaceCanBeWritten(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})

	for _, path := range []string{"/tmp/dcode-eval-escape.txt", "../escaped.txt"} {
		out, isErr := w.Execute(context.Background(), "write",
			[]byte(`{"path":"`+path+`","content":"escaped"}`))
		if !isErr {
			t.Errorf("writing %s succeeded: %q", path, out)
		}
	}
}

// Inside still works, or the gate has replaced one broken measurement with
// another.
func TestInsideTheWorkspaceStillRuns(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})
	if out, isErr := w.Execute(context.Background(), "read", []byte(`{"path":"stats.go"}`)); isErr {
		t.Errorf("a read inside the workspace was refused: %q", out)
	}
	if out, isErr := w.Execute(context.Background(), "write",
		[]byte(`{"path":"new.go","content":"package stats\n"}`)); isErr {
		t.Errorf("a write inside the workspace was refused: %q", out)
	}
}
