package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

// readKeyCall matches a typed read of a configuration key on a Resolved:
// r.Bool("memory.enabled", …), resolved.String("ui.lang", …), r.Get("…").
var readKeyCall = regexp.MustCompile(`\b[A-Za-z_]\w*\.(?:String|Int|Bool|Get)\("([a-z_]+\.[a-z_]+)"`)

// TestEveryKeyReadIsAKnownKey is the reverse of TestEveryKnownKeyIsAccountedFor.
//
// That guard starts from KnownKeys and proves each one is read. It cannot see
// a key the code reads that never made it into KnownKeys: the read compiles,
// returns its default, and a config.toml that writes the key is refused as
// unknown. memory.enabled and memory.max_entries lived exactly there.
//
// Starting from the sources instead closes the other direction: every key any
// product source reads by name must be in the schema.
func TestEveryKeyReadIsAKnownKey(t *testing.T) {
	read := keysReadBySources(t, filepath.Join("..", "..", "internal"), filepath.Join("..", "..", "cmd"))
	if len(read) == 0 {
		t.Fatal("no key reads found; the guard would pass vacuously")
	}
	for key, where := range read {
		if _, ok := config.KnownKeys[key]; !ok {
			t.Errorf("%s reads %q, which is not in KnownKeys: config.toml refuses it "+
				"and its environment variable is never looked up", where, key)
		}
	}
}

// keysReadBySources maps each key read in the non-test Go sources under roots
// to the first file that reads it.
func keysReadBySources(t *testing.T, roots ...string) map[string]string {
	t.Helper()
	read := map[string]string{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, m := range readKeyCall.FindAllSubmatch(data, -1) {
				key := string(m[1])
				if _, seen := read[key]; !seen {
					read[key] = path
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	return read
}
