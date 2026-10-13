package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The id a session continues is a file name inside the record directory and
// nothing more. Joined to the path unchecked, `../secret` continued a record one
// directory up: read into the model's context and copied into the new session's
// own record.
func TestContinuingAnIdThatLeavesTheRecordDirectoryIsRefused(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sessions")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"seq":1,"type":"session.created","at":"2026-08-14T12:00:00Z","payload":{"id":"secret","workspace":"/w"}}
{"seq":2,"type":"turn.started","at":"2026-08-14T12:00:01Z","payload":{"turn_id":"t1","text":"não devia sair"}}
{"seq":3,"type":"turn.completed","at":"2026-08-14T12:00:03Z","payload":{"turn_id":"t1"}}
`
	if err := os.WriteFile(filepath.Join(root, "secret.jsonl"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		RecordDir:      dir,
		Base:           baseOpts(t),
	})

	for _, id := range []string{"..", "../secret", "a/b", `a\b`, filepath.Join(root, "secret"), "secret.jsonl"} {
		sess, err := d.build(protocol.CreateSessionRequest{
			Workspace: t.TempDir(), Model: "MiniMax-M3", Resume: id,
		})
		if sess != nil {
			sess.Close()
		}
		if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeInvalidInput {
			t.Errorf("Resume %q: got %v, want %s", id, err, protocol.CodeInvalidInput)
		}
	}
}
