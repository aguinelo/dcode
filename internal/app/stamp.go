package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/config"
)

// GeneratedFile is the one file dcode writes on the user's behalf and then
// hands over. Everything else in the workspace is theirs from the start.
const GeneratedFile = "DCODE.md"

// StampGenerated records, inside a freshly generated DCODE.md, the digests of
// the instruction files it was generated from.
//
// Without the record the divergence warning cannot exist. `/init` reads
// AGENTS.md and writes a DCODE.md from it; months later AGENTS.md changes, and
// the DCODE.md still says what the old one said — silently, because a generated
// file that has since been hand-edited looks exactly like one that is still
// current. RenderDigest, Diverged and the warning itself were all written; the
// marker was never put in the file, so Diverged could only ever answer "nothing
// changed".
//
// Deterministic, and done by the product rather than asked of the model. A
// prompt requesting the marker would be a prompt hoping for it, and the digests
// have to be of the bytes actually read.
//
// It runs after a turn, so failure here must stay quiet. This is bookkeeping,
// and a session that breaks because a comment could not be appended has traded
// a missing warning for a broken tool.
func StampGenerated(workspace string, sources, written []string) {
	if len(sources) == 0 || !wrote(written, GeneratedFile) {
		return
	}
	path := filepath.Join(workspace, GeneratedFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Stamped once, on the turn that generated it. From then the file belongs
	// to the human — re-stamping would rewrite what they have edited since, and
	// would move the baseline forward, turning "these sources changed" into
	// "nothing changed": the exact warning this exists to make possible.
	body := string(data)
	if strings.Contains(body, config.DigestMarker) {
		return
	}

	marker := config.RenderDigest(os.DirFS(workspace), sources)
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	// Same permissions the model wrote it with; this is an append, not a new
	// file, and widening them silently would be a change nobody asked for.
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, []byte(body+"\n"+marker+"\n"), info.Mode().Perm())
}

// wrote reports whether name is among the paths a turn changed. Compared by
// base name because a written path is workspace-relative and the generated file
// only ever lives at the root.
func wrote(written []string, name string) bool {
	for _, w := range written {
		if w == name || filepath.Base(w) == name && filepath.Dir(w) == "." {
			return true
		}
	}
	return false
}
