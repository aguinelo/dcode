package app

import (
	"os"
	"path/filepath"
	"reflect"
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
