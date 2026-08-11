package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// specRow matches a key declared in a `.config.spec.md` table.
//
// Only table rows. A name appearing in prose — typically a note saying a key
// was removed — is documentation about the past, not a declaration.
var specRow = regexp.MustCompile("(?m)^\\| `(DCODE_[A-Z_]+)` \\|")

// The guard that closes the loop, and the one that did not exist.
//
// TestEveryKnownKeyIsAccountedFor watches KnownKeys and catches a key that is
// declared in code and read by nothing. It never reached the other side: a key
// the SPEC declares which never made it into KnownKeys fails no test at all,
// and goes on looking like configuration to whoever reads the spec.
//
// Measured on 2026-08-11, before this ran: of 112 keys declared in tables
// across the architecture specs, 64 were referenced nowhere in the tree. A
// declared configuration surface that does not exist is worse than an absent
// one, because it promises a control that is not there — the value is read,
// resolved, shown by `dcode config`, and ignored.
// declaredNotYetRead is the exception list, and it is meant to reach zero.
//
// Same shape as nonSession in the wiring guard: an escape hatch is only
// tolerable when it is named, explained, and small enough to read. An entry
// here is a promise that the key is being implemented, not that it is exempt.
var declaredNotYetRead = map[string]string{
	"DCODE_LANG": "the interface language; RN-19 of client-tui is specified and the catalogue is not built yet",
}

func TestEveryKeyTheSpecsDeclareIsReadSomewhere(t *testing.T) {
	root := repoRoot(t)
	src := goSources(t, root)

	specs, err := filepath.Glob(filepath.Join(root, "docs", "specs", "architecture", "*", "*.config.spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) == 0 {
		t.Fatal("no config specs found; the guard would pass vacuously")
	}

	for _, spec := range specs {
		data, err := os.ReadFile(spec)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range specRow.FindAllStringSubmatch(string(data), -1) {
			key := m[1]
			if _, ok := EnvToKey[key]; ok {
				continue
			}
			if strings.Contains(src, key) {
				continue
			}
			if _, ok := declaredNotYetRead[key]; ok {
				continue
			}
			t.Errorf("%s declares %s, which is in neither KnownKeys nor any source. "+
				"Either wire it in the same change that declares it, or take the row out.",
				filepath.Base(spec), key)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// goSources concatenates the non-test Go, shell and workflow files, which is
// where a key would be read if it were read at all.
//
// Tests are excluded on purpose, and it makes both guards stricter: a
// configuration key mentioned only from a test is not implemented, and without
// the exclusion this file's own exception list would count as an
// implementation of the keys it excuses.
func goSources(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "bin", "docs":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		switch {
		case strings.HasSuffix(path, ".go"),
			strings.HasSuffix(path, ".sh"),
			strings.HasSuffix(path, ".yml"),
			d.Name() == "Makefile":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b.Write(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.Len() == 0 {
		t.Fatal("no sources read; the guard would pass vacuously")
	}
	return b.String()
}

// The other half of the exception list. An entry that has been implemented, or
// that never existed, is an entry nobody will notice has gone stale.
func TestTheExceptionListIsStillAccurate(t *testing.T) {
	root := repoRoot(t)
	src := goSources(t, root)

	for key, why := range declaredNotYetRead {
		if _, ok := EnvToKey[key]; ok {
			t.Errorf("%s is excused as %q and is now in KnownKeys; take it off the list", key, why)
		}
		if strings.Contains(src, key) {
			t.Errorf("%s is excused as %q and is now read in code; take it off the list", key, why)
		}
	}
}
