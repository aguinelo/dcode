package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
)

// opening is one thing a session said as it opened, read the way an attached
// client reads it: by the names on the wire, not by this package's types.
type opening struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// openingNotices opens a session through the daemon, attaches to it, and
// returns every notice in its log up to the sequence the creation response
// reported — everything an attached client reads before the first turn.
func openingNotices(t *testing.T, ws string, req protocol.CreateSessionRequest) []opening {
	t.Helper()
	_, c, _ := daemonFor(t, ws)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.Workspace = ws
	sess, err := c.CreateSession(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	events, errs := c.Subscribe(ctx, sess.ID, 1)
	var got []opening
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatalf("the stream ended before seq %d", sess.LastSeq)
			}
			if ev.Type == "session.notice" {
				var n opening
				if err := json.Unmarshal(ev.Payload, &n); err != nil {
					t.Fatalf("a notice that does not decode: %v", err)
				}
				got = append(got, n)
			}
			if ev.Seq >= sess.LastSeq {
				return got
			}
		case err, ok := <-errs:
			if ok && err != nil {
				t.Fatal(err)
			}
			errs = nil
		case <-ctx.Done():
			t.Fatalf("the events up to seq %d never arrived", sess.LastSeq)
		}
	}
}

// RN-11 of provider-adapter, where a person can see it.
//
// A family with no measurements behind it says so in the session, and until
// this it did not, twice over. New built the family's admission and then put
// the instruction notice in its place — "" in a workspace with nothing to
// translate, so the default configuration said nothing — and nothing read
// what was left, so no configuration said anything either. The guard on the
// list of who warns stayed green throughout: it checked the list, and the
// list was right.
//
// generic through a profile, because no model name resolves to it: it is the
// explicit hatch, and a profile is how a person names it.
func TestAnUnmeasuredFamilySaysSoToWhoeverAttaches(t *testing.T) {
	for _, tc := range []struct {
		family, model string
		// models is the project's models.toml, when one is needed.
		models string
	}{
		{family: provider.ClaudeName, model: "claude-sonnet-4-5"},
		{family: provider.GenericName, model: "local",
			models: "[profile.local]\nmodel = \"qwen3.5-9b\"\nfamily = \"generic\"\n"},
	} {
		t.Run(tc.family, func(t *testing.T) {
			want := provider.Unmeasured(tc.family)
			if want == "" {
				t.Fatalf("%s has measurements now; this needs a family that has none", tc.family)
			}
			ws := t.TempDir()
			if tc.models != "" {
				if err := os.MkdirAll(filepath.Join(ws, ".dcode"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(ws, ".dcode", "models.toml"), []byte(tc.models), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			got := openingNotices(t, ws, protocol.CreateSessionRequest{Model: tc.model})
			for _, n := range got {
				if n.Code == "family_unmeasured" && n.Message == want {
					return
				}
			}
			t.Errorf("a session on the %s family opened without saying it is unmeasured; "+
				"an attached client read %+v", tc.family, got)
		})
	}
}

// The instruction notice does not take the family's place.
//
// They are about the same session and answer different questions — which
// model this is, and whose instructions it is reading — so neither is the
// other's to replace. One notice each, the family's first: it is about the
// whole session, and the instructions are about one input to it.
func TestNoOpeningNoticeTakesAnothersPlace(t *testing.T) {
	ws := t.TempDir()
	// Written for another tool, and no DCODE.md beside it: the case the
	// instruction notice exists for. It names no tool and no command, so the
	// session's tool list cannot change what the notice counts, and the text
	// can be asked of InstructionNotice rather than copied here.
	if err := os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte("Answer in English.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := openingNotices(t, ws, protocol.CreateSessionRequest{Model: "claude-sonnet-4-5"})
	var codes []string
	for _, n := range got {
		codes = append(codes, n.Code)
	}
	if want := []string{"family_unmeasured", "instructions_untranslated"}; !slices.Equal(codes, want) {
		t.Fatalf("the session opened saying %q, want %q: %+v", codes, want, got)
	}
	if want := provider.Unmeasured(provider.ClaudeName); got[0].Message != want {
		t.Errorf("the family said %q, want %q", got[0].Message, want)
	}
	if want := InstructionNotice(ws, ForeignDefault, nil); got[1].Message != want {
		t.Errorf("the instructions said %q, want %q", got[1].Message, want)
	}
}
