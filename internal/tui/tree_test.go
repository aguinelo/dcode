package tui

import (
	"reflect"
	"testing"
)

func tool(t, target string, running, failed bool, added int) Entry {
	return Entry{Kind: KindTool, Tool: t, Target: target, Running: running, IsError: failed, Added: added}
}

func labels(rows []FileRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Label)
	}
	return out
}

// The invariant the design states first, and the one a sidebar gets wrong in
// the way that matters: a file the turn has not created is not drawn. A list
// that promises a path which is not on disk sends someone looking for it.
func TestAPathNoToolTouchedIsNotDrawn(t *testing.T) {
	rows := FileTree([]Entry{
		tool("read", "internal/tui/render.go", false, false, 0),
		{Kind: KindAssistant, Summary: "I will create internal/tui/rail.go next"},
	})
	for _, r := range rows {
		if r.Path == "internal/tui/rail.go" {
			t.Errorf("a file only mentioned in prose was drawn: %+v", rows)
		}
	}
}

// A pattern is not a path, and a command line is not a path. Drawing either
// puts something in the list that cannot be opened.
func TestPatternsAndCommandsStayOutOfTheFileList(t *testing.T) {
	rows := FileTree([]Entry{
		tool("grep", `\.Save\(`, false, false, 0),
		tool("bash", "go test ./...", false, false, 0),
		tool("read", "go.mod", false, false, 0),
	})
	if got := labels(rows); !reflect.DeepEqual(got, []string{"go.mod"}) {
		t.Errorf("a pattern or a command reached the file list: %v", got)
	}
}

// A folder with a single child is one row. Two rows for a directory nobody
// needs to see on its own is how a narrow column runs out of height.
func TestAFolderWithASingleChildIsCompacted(t *testing.T) {
	rows := FileTree([]Entry{tool("read", "internal/tui/render.go", false, false, 0)})
	want := []string{"internal/tui/", "render.go"}
	if got := labels(rows); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Siblings share the folder they are in rather than repeating it.
func TestSiblingsShareTheirFolder(t *testing.T) {
	rows := FileTree([]Entry{
		tool("read", "internal/tui/render.go", false, false, 0),
		tool("edit", "internal/tui/model.go", false, false, 0),
	})
	want := []string{"internal/tui/", "model.go", "render.go"}
	if got := labels(rows); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// The last event to touch a path is what the row says about it. A file read and
// then written is being written.
func TestTheLastEventDecidesTheState(t *testing.T) {
	rows := FileTree([]Entry{
		tool("read", "a.go", false, false, 0),
		tool("edit", "a.go", true, false, 0),
	})
	if len(rows) != 1 || rows[0].State != FileWriting {
		t.Errorf("the later write did not win: %+v", rows)
	}
}

// A call that came back an error marks the path failed, whatever ran before it.
func TestAFailedCallMarksThePath(t *testing.T) {
	rows := FileTree([]Entry{
		tool("read", "a.go", false, false, 0),
		tool("write", "a.go", false, true, 0),
	})
	if len(rows) != 1 || rows[0].State != FileFailed {
		t.Errorf("the failure did not reach the row: %+v", rows)
	}
}

// The line count comes from the number the tool reported, never from reading it
// back out of the summary sentence.
func TestTheCountComesFromTheToolAndNotFromItsSentence(t *testing.T) {
	rows := FileTree([]Entry{tool("write", "new.go", false, false, 38)})
	if len(rows) != 1 || rows[0].Added != 38 {
		t.Errorf("the reported count did not reach the row: %+v", rows)
	}
}

// Same entries, same tree — which is what makes reopening a session reproduce
// the sidebar it had. Derived rather than stored is what buys this.
func TestTheSameEntriesProduceTheSameTree(t *testing.T) {
	entries := []Entry{
		tool("edit", "b/y.go", false, false, 2),
		tool("read", "a/x.go", false, false, 0),
		tool("read", "b/z.go", true, false, 0),
	}
	if !reflect.DeepEqual(FileTree(entries), FileTree(entries)) {
		t.Error("two derivations of one entry list disagreed")
	}
}

// Sorted, so the list does not reshuffle under the reader while work goes on.
func TestTheListIsSortedRatherThanInTouchOrder(t *testing.T) {
	rows := FileTree([]Entry{
		tool("read", "z.go", false, false, 0),
		tool("read", "a.go", false, false, 0),
	})
	if got := labels(rows); !reflect.DeepEqual(got, []string{"a.go", "z.go"}) {
		t.Errorf("the list follows touch order and will jump around: %v", got)
	}
}

func TestATurnThatTouchedNothingHasNoTree(t *testing.T) {
	if rows := FileTree(nil); rows != nil {
		t.Errorf("an empty turn produced rows: %+v", rows)
	}
}
