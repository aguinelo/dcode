package tools

import "testing"

// Regression: bash asked for network approval on every command, in a
// configuration where the sandbox blocks the network at the OS level.
//
// The prompt said "this would reach the network" about a command that could
// not. Approving granted nothing — the command still could not resolve a host —
// and denying stopped the whole command rather than its network access. The
// user was answering a question that was not the one on screen.
//
// A crossing the mechanism already prevents is a false alarm, not honesty.
func TestBashDeclaresNetworkOnlyWhenTheSandboxWouldPermitIt(t *testing.T) {
	for _, tc := range []struct {
		name    string
		allowed bool
	}{
		{"blocked by the sandbox", false},
		{"permitted by the sandbox", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := Bash{AllowNetwork: tc.allowed}
			req, err := b.Declare([]byte(`{"command":"go test ./..."}`))
			if err != nil {
				t.Fatal(err)
			}
			if req.Network != tc.allowed {
				t.Errorf("declared network %v while the sandbox permits %v",
					req.Network, tc.allowed)
			}
			// What is opaque about a shell command has not changed: it can
			// still write, and the workspace boundary is what answers that.
			if len(req.Paths) == 0 || !req.Paths[0].Write {
				t.Error("a shell command must still declare that it writes")
			}
			if req.Command == "" {
				t.Error("the command must reach the approval, or consent is blind")
			}
		})
	}
}
