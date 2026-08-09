package policy

import "testing"

// A rule that cannot say "anywhere under this directory" is a rule people work
// around by listing every depth, so `**` has to work and has to be exact.
func TestGlob(t *testing.T) {
	for _, tc := range []struct {
		pattern, s string
		want       bool
	}{
		// A single star stays inside one segment.
		{"*.go", "main.go", true},
		{"*.go", "src/main.go", false},
		{"src/*.go", "src/main.go", true},
		{"src/*.go", "src/deep/main.go", false},

		// A double star crosses them.
		{"**/*.go", "src/deep/main.go", true},
		{"**/*.go", "main.go", true},
		{".git/**", ".git/config", true},
		{".git/**", ".git/hooks/pre-commit", true},
		{".git/**", "src/.git/config", false},
		{"**/.env", ".env", true},
		{"**/.env", "services/api/.env", true},
		{"**/.env", ".env.local", false},

		// A question mark is one character, and not a separator.
		{"id_?sa", "id_rsa", true},
		{"a?c", "a/c", false},

		// Exact is exact.
		{".env", ".env", true},
		{".env", ".env.local", false},
		{".env.*", ".env.local", true},

		// Nothing matches nothing.
		{"", "", true},
		{"*", "", true},
		{"a", "", false},
	} {
		if got := Glob(tc.pattern, tc.s); got != tc.want {
			t.Errorf("Glob(%q, %q) = %v", tc.pattern, tc.s, got)
		}
	}
}

// The path is workspace-relative with forward slashes, so a rule reads the same
// on every platform and cannot be dodged by an absolute spelling.
func TestMatchPathNormalisesBeforeComparing(t *testing.T) {
	r := Rules{ConfirmWrite: []string{".git/**"}}
	for _, p := range []string{".git/config", "./.git/config", "/.git/config", `.git\config`} {
		if _, ok := r.MatchPath(p, true); !ok {
			t.Errorf("%q should have matched", p)
		}
	}
}

// Read and write are separate lists: reading a secret and writing a hook are
// different questions, and conflating them asks the wrong one.
func TestReadAndWriteRulesAreSeparate(t *testing.T) {
	r := Rules{
		ConfirmWrite: []string{".git/**"},
		ConfirmRead:  []string{".env"},
	}
	if _, ok := r.MatchPath(".git/config", true); !ok {
		t.Error("writing .git must ask")
	}
	if _, ok := r.MatchPath(".git/config", false); ok {
		t.Error("reading .git is not the same question")
	}
	if _, ok := r.MatchPath(".env", false); !ok {
		t.Error("reading .env must ask")
	}
	if _, ok := r.MatchPath(".env", true); ok {
		t.Error("writing .env was not asked about")
	}
}

// The matched pattern comes back, because the modal has to say which rule
// fired and the session grant is keyed on it.
func TestMatchReportsWhichRuleFired(t *testing.T) {
	r := Rules{ConfirmWrite: []string{"*.lock", ".git/**"}}
	got, ok := r.MatchPath(".git/config", true)
	if !ok || got != ".git/**" {
		t.Errorf("got %q, %v", got, ok)
	}
	if _, ok := r.MatchPath("src/main.go", true); ok {
		t.Error("an ordinary source file must not ask")
	}
}

func TestMatchCommand(t *testing.T) {
	r := Rules{ConfirmCommand: []string{"rm -rf*", "git push --force*"}}
	for _, tc := range []struct {
		cmd  string
		want bool
	}{
		{"rm -rf build/", true},
		{"  rm -rf /  ", true},
		{"git push --force-with-lease", true},
		{"go test ./...", false},
		{"", false},
	} {
		if _, ok := r.MatchCommand(tc.cmd); ok != tc.want {
			t.Errorf("%q: got %v", tc.cmd, ok)
		}
	}
}

// Every default earns its place by being different in kind from an ordinary
// source file, not merely important.
func TestDefaultRulesCoverTheEscapeAndTheSecrets(t *testing.T) {
	r := DefaultRules()

	// A write to .git/hooks runs on the next commit, outside the sandbox.
	if _, ok := r.MatchPath(".git/hooks/pre-commit", true); !ok {
		t.Error("a git hook must ask before being written")
	}
	// An agent that edits its own configuration can widen its own reach.
	if _, ok := r.MatchPath(".dcode/config.toml", true); !ok {
		t.Error("the agent's own configuration must ask")
	}
	if _, ok := r.MatchPath(".dcode/commands/x.md", true); !ok {
		t.Error("a user command is configuration too")
	}
	// Reading a secret sends it to the model provider, off this machine.
	for _, secret := range []string{
		".env", ".env.local", "services/api/.env",
		"certs/server.pem", "deploy/id_rsa", ".npmrc",
	} {
		if _, ok := r.MatchPath(secret, false); !ok {
			t.Errorf("reading %q must ask", secret)
		}
	}

	// And ordinary work stays out of the way, which is the whole point of the
	// list being short.
	for _, ordinary := range []string{
		"src/main.go", "README.md", "internal/policy/rules.go", "go.mod",
	} {
		if _, ok := r.MatchPath(ordinary, true); ok {
			t.Errorf("writing %q must not ask", ordinary)
		}
		if _, ok := r.MatchPath(ordinary, false); ok {
			t.Errorf("reading %q must not ask", ordinary)
		}
	}

	// No command rules by default: the sandbox contains, and a list that pauses
	// on someone else's idea of dangerous is noise until it is configured.
	if len(r.ConfirmCommand) != 0 {
		t.Errorf("got %v", r.ConfirmCommand)
	}
}

func TestListRoundTrip(t *testing.T) {
	in := []string{".git/**", ".dcode/**"}
	if got := SplitList(JoinList(in)); len(got) != 2 || got[0] != in[0] {
		t.Errorf("got %v", got)
	}
	if got := SplitList(" a , , b "); len(got) != 2 || got[1] != "b" {
		t.Errorf("blanks must be dropped, got %v", got)
	}
	if got := SplitList(""); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

// A pattern that is only whitespace matches nothing rather than everything —
// the failure mode where a stray comma in configuration gates the whole
// workspace.
func TestABlankPatternMatchesNothing(t *testing.T) {
	r := Rules{ConfirmWrite: []string{"", "   "}}
	if _, ok := r.MatchPath("src/main.go", true); ok {
		t.Error("a blank pattern must not claim everything")
	}
}
