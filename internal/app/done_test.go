package app

import (
	"context"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/sandbox"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
)

func TestNoDoneFileAndNoVerifyCommandMeansNoCriteria(t *testing.T) {
	set, err := loadDoneSet(filepath.Join(t.TempDir(), "done.toml"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 0 {
		t.Fatalf("criteria = %v, want none", set.Criteria)
	}
}

// The two mechanisms do not coexist: a verify command IS a set of one.
func TestAVerifyCommandBecomesASetOfOne(t *testing.T) {
	set, err := loadDoneSet(filepath.Join(t.TempDir(), "done.toml"), "make check")
	if err != nil {
		t.Fatal(err)
	}
	want := []loop.Criterion{{Name: "verify", Command: "make check"}}
	if !reflect.DeepEqual(set.Criteria, want) {
		t.Fatalf("criteria = %v, want %v", set.Criteria, want)
	}
}

func TestTheDoneFileDeclaresCriteriaAndProtectedPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "done.toml")
	if err := os.WriteFile(path, []byte(`
protected = ["**/*_test.go", "testdata/**"]

[tests]
command = "make test"

[no-todos]
command = "grep -rq TODO ."
exit_code = 1
`), 0o644); err != nil {
		t.Fatal(err)
	}

	set, err := loadDoneSet(path, "make check")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 2 {
		t.Fatalf("criteria = %v, want two", set.Criteria)
	}
	// The file wins over the single verify command; they are not merged, and
	// they are not two mechanisms.
	if set.Criteria[0].Name != "tests" || set.Criteria[0].Command != "make test" {
		t.Errorf("first criterion = %+v", set.Criteria[0])
	}
	if set.Criteria[1].ExitCode != 1 {
		t.Errorf("a criterion met by exit 1 lost its exit code: %+v", set.Criteria[1])
	}
	if !reflect.DeepEqual(set.Protected, []string{"**/*_test.go", "testdata/**"}) {
		t.Errorf("protected = %v", set.Protected)
	}
}

// Under .dcode/, which DefaultRules already submits to write confirmation. An
// agent that can quietly edit its own definition of done widens its own reach.
func TestTheDoneFileLivesUnderDotDcode(t *testing.T) {
	got := doneFilePath("", "/w")
	want := filepath.Join("/w", ".dcode", "done.toml")
	if got != want {
		t.Fatalf("done file = %q, want %q", got, want)
	}
}

func TestAMalformedDurationFallsBackRatherThanFailing(t *testing.T) {
	for _, in := range []string{"", "banana", "-5m", "0"} {
		if got := parseDuration(in, time.Minute); got != time.Minute {
			t.Errorf("parseDuration(%q) = %v, want the fallback", in, got)
		}
	}
	if got := parseDuration("30s", time.Minute); got != 30*time.Second {
		t.Errorf("a valid duration was not honoured: %v", got)
	}
}

// The delegating tool is wired to the engine that owns turns, and the child it
// builds is read-only by construction. Registering it and forgetting the knot
// would leave a tool that always errors.
func TestExploreIsWiredToTheEngine(t *testing.T) {
	e := &loop.Engine{}
	var _ interface {
		Explore(ctx context.Context, task, path string) (string, []string, []string, bool, error)
	} = e
}

// A criterion command goes through the sandbox, not around it. It comes from
// configuration a person reviewed — which is why it may run at all, since
// RN-6.1 forbids running one read from a shared instruction file — but
// "reviewed" is not "unconfined".
func TestACriterionRunsInsideTheSandbox(t *testing.T) {
	sb, err := sandbox.New(sandbox.Config{Backend: sandbox.BackendAuto}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no sandbox available here: %v", err)
	}
	ws := t.TempDir()
	run := criterionRunner(sb, Options{
		Workspace:   ws,
		SandboxMode: policy.ModeWorkspaceWrite,
		DoneTimeout: 30 * time.Second,
	})

	code, out, err := run(context.Background(), "printf ok")
	if err != nil {
		t.Fatalf("a trivial command failed to run: %v", err)
	}
	if code != 0 {
		t.Errorf("exit = %d, want 0 (output %q)", code, out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("output = %q", out)
	}

	// A non-zero exit is a verdict, not a failure to run: the criterion is
	// unmet, and Check must be able to tell the two apart.
	code, _, err = run(context.Background(), "exit 3")
	if err != nil {
		t.Fatalf("a failing command reported an error rather than its code: %v", err)
	}
	if code != 3 {
		t.Errorf("exit = %d, want 3", code)
	}
}
