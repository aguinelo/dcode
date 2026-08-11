package tools

import "testing"

// A shell command is opaque, so the worst case is what gets declared. There is
// no reading of the string that answers whether this one reaches out: a build
// resolves dependencies, a test suite pulls an image, a formatter checks for a
// newer version.
//
// This once depended on what the sandbox was built to permit, and the reasoning
// was sound then: the OS blocked the network either way, so approving granted
// nothing and denying stopped the whole command rather than its network access
// — the user answered a question that was not the one on screen.
//
// That premise is gone. The sandbox is asked per command and a grant opens it,
// so consent now means what it appears to mean, and the question is asked once
// per project instead of once per command.
func TestBashAlwaysDeclaresThatACommandMayReachTheNetwork(t *testing.T) {
	for _, command := range []string{
		"go test ./...", "ls", "echo hello", "curl https://example.com",
	} {
		req, err := Bash{}.Declare([]byte(`{"command":"` + command + `"}`))
		if err != nil {
			t.Fatal(err)
		}
		if !req.Network {
			t.Errorf("%q was declared unable to reach the network; nothing here can know that", command)
		}
		// What is opaque about a shell command has not changed: it can still
		// write, and the workspace boundary is what answers that.
		if len(req.Paths) == 0 || !req.Paths[0].Write {
			t.Error("a shell command must still declare that it writes")
		}
		if req.Command == "" {
			t.Error("the command must reach the approval, or consent is blind")
		}
	}
}
