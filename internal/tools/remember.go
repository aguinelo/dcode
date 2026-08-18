package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/memory"
	"github.com/aguinelo/dcode/internal/policy"
)

// Remember writes something learned where the next session will read it.
//
// A tool rather than something that happens on its own, and the distinction is
// the design: reading what was learned is a fact the model needs every turn, so
// it lives in the prefix; writing one is an act with consequences, and an act
// that happens without being asked is an act nobody authorised.
type Remember struct {
	// Commit and Today are handed in rather than read here, so the provenance a
	// memory carries agrees with the repository state the prefix already
	// described. Two readings of git in one session can disagree, and
	// disagreeing about where the session is is worse than not knowing.
	Commit string
	Today  string
}

// RememberInput is the argument shape.
type RememberInput struct {
	Kind    string `json:"kind"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (Remember) Name() string { return "remember" }

func (Remember) Description() string {
	return "Write down something this repository taught you, for the next session to read. " +
		"Use it for what cost time and will cost it again (gotcha), a choice and its reason " +
		"so nobody relitigates it (decision), or how this repository does something that is " +
		"nowhere written down (convention). " +
		"NOT for what you did — that is already recorded, and a memory of activity is noise " +
		"by next week. It lands in the next session, never this one, and a person reviews it " +
		"in the diff like any other change."
}

func (Remember) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"kind":{"type":"string","enum":["gotcha","decision","convention"],` +
		`"description":"gotcha: cost time and will again. decision: a choice and why. ` +
		`convention: how this repository does something."},` +
		`"subject":{"type":"string","description":"One line somebody would search for."},` +
		`"body":{"type":"string","description":"What to know, in a sentence or three."}},` +
		`"required":["kind","subject","body"]}`)
}

// Declare reports the one path it writes and nothing else.
func (r Remember) Declare(input json.RawMessage) (policy.Request, error) {
	var in RememberInput
	if err := decode(r.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	return policy.Request{
		Tool:  r.Name(),
		Paths: []policy.Access{{Path: memory.FileName, Write: true}},
	}, nil
}

func (r Remember) Execute(_ context.Context, input json.RawMessage, s *State) (Result, error) {
	var in RememberInput
	if err := decode(r.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}

	kind := memory.Kind(strings.ToLower(strings.TrimSpace(in.Kind)))
	if !kind.Valid() {
		// Naming them: "invalid kind" leaves the model guessing at a list it
		// cannot see.
		return errf(r.Name(), CodeBadInput,
			fmt.Sprintf("Use one of: %s.", strings.Join(kindNames(), ", ")),
			"%q is not a kind of memory", in.Kind).Result(), nil
	}
	subject := strings.TrimSpace(in.Subject)
	if subject == "" {
		return errf(r.Name(), CodeBadInput,
			"Give one line somebody would search for.",
			"a memory with no subject is a memory nobody finds").Result(), nil
	}

	abs, terr := resolvePath(r.Name(), s, memory.FileName, true)
	if terr != nil {
		return terr.Result(), nil
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return errf(r.Name(), CodeNotFound, "", "could not create %s: %v",
			filepath.Dir(memory.FileName), err).Result(), nil
	}

	// Appended, never rewritten. Sorting, deduping or tidying would turn a
	// three-line change into a whole-file diff, and an unreadable diff is a
	// review that does not happen — which is the only quality gate this has.
	f, err := os.OpenFile(abs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return errf(r.Name(), CodeNotFound, "", "could not write %s: %v",
			memory.FileName, err).Result(), nil
	}
	block := r.render(kind, subject, strings.TrimSpace(in.Body))
	if _, err := f.WriteString(block); err != nil {
		f.Close()
		return errf(r.Name(), CodeNotFound, "", "could not write %s: %v",
			memory.FileName, err).Result(), nil
	}
	if err := f.Close(); err != nil {
		return errf(r.Name(), CodeNotFound, "", "could not write %s: %v",
			memory.FileName, err).Result(), nil
	}

	s.MarkWritten(abs)

	return Result{
		Output: fmt.Sprintf("remembered as %s: %s\n\n"+
			"It goes in %s and reaches the model NEXT session — this session's prefix was "+
			"fixed when it opened. Somebody reviews it in the diff.",
			kind, subject, memory.FileName),
		Meta: Meta{Files: 1},
	}, nil
}

// render writes one block in the grammar the reader parses.
func (r Remember) render(kind memory.Kind, subject, body string) string {
	var b strings.Builder
	b.WriteString("\n## " + string(kind) + ": " + subject + "\n")

	// Provenance only when there is any. A workspace that is not a repository
	// has no commit to name, and inventing one would be worse than the absence.
	var parts []string
	if r.Today != "" {
		parts = append(parts, "learned "+r.Today)
	}
	if r.Commit != "" {
		parts = append(parts, "commit "+r.Commit)
	}
	if len(parts) > 0 {
		b.WriteString("<!-- " + strings.Join(parts, " · ") + " -->\n")
	}
	if body != "" {
		b.WriteString("\n" + body + "\n")
	}
	return b.String()
}

func kindNames() []string {
	var out []string
	for _, k := range memory.Kinds() {
		out = append(out, string(k))
	}
	return out
}
