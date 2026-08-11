package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// One chain for the whole product. An assertion per adjacent pair, because a
// value that resolves differently than documented is the kind of bug that
// produces "it works on my machine".
func TestPrecedenceChain(t *testing.T) {
	for _, tc := range []struct {
		name    string
		layers  []Layer
		want    string
		wantSrc Source
	}{
		{"default alone", []Layer{
			{Source: SourceDefault, Origin: "built-in", Values: map[string]string{"k": "d"}},
		}, "d", SourceDefault},

		{"user beats default", []Layer{
			{Source: SourceDefault, Origin: "built-in", Values: map[string]string{"k": "d"}},
			{Source: SourceUser, Origin: "~/config.toml", Values: map[string]string{"k": "u"}},
		}, "u", SourceUser},

		{"project beats user", []Layer{
			{Source: SourceUser, Origin: "~/config.toml", Values: map[string]string{"k": "u"}},
			{Source: SourceProject, Origin: ".dcode/config.toml", Values: map[string]string{"k": "p"}},
		}, "p", SourceProject},

		{"env beats project", []Layer{
			{Source: SourceProject, Origin: ".dcode", Values: map[string]string{"k": "p"}},
			{Source: SourceEnv, Origin: "DCODE_K", Values: map[string]string{"k": "e"}},
		}, "e", SourceEnv},

		{"flag beats env", []Layer{
			{Source: SourceEnv, Origin: "DCODE_K", Values: map[string]string{"k": "e"}},
			{Source: SourceFlag, Origin: "--k", Values: map[string]string{"k": "f"}},
		}, "f", SourceFlag},

		{"locked beats flag", []Layer{
			{Source: SourceFlag, Origin: "--k", Values: map[string]string{"k": "f"}},
			{Source: SourceLocked, Origin: "requirements.toml", Values: map[string]string{"k": "l"}},
		}, "l", SourceLocked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Resolve(tc.layers).Get("k")
			if !ok {
				t.Fatal("key not resolved")
			}
			if got.Value != tc.want || got.Source != tc.wantSrc {
				t.Errorf("got %q from %s, want %q from %s",
					got.Value, got.Source, tc.want, tc.wantSrc)
			}
		})
	}
}

// Layer order in the slice must not matter: only the declared rank does.
func TestResolveIgnoresLayerOrdering(t *testing.T) {
	a := Layer{Source: SourceLocked, Origin: "req", Values: map[string]string{"k": "l"}}
	b := Layer{Source: SourceEnv, Origin: "env", Values: map[string]string{"k": "e"}}

	first, _ := Resolve([]Layer{a, b}).Get("k")
	second, _ := Resolve([]Layer{b, a}).Get("k")
	if first.Value != second.Value {
		t.Errorf("order changed the result: %q vs %q", first.Value, second.Value)
	}
}

// Ignoring an override in silence leaves the user believing the change took
// effect, which is the more expensive failure.
func TestLockedOverrideIsWarnedAboutNotSwallowed(t *testing.T) {
	r := Resolve([]Layer{
		{Source: SourceLocked, Origin: "/etc/dcode/requirements.toml",
			Values: map[string]string{"sandbox.mode": "workspace-write"}},
		{Source: SourceEnv, Origin: "DCODE_SANDBOX_MODE",
			Values: map[string]string{"sandbox.mode": "full-access"}},
	})

	v, _ := r.Get("sandbox.mode")
	if v.Value != "workspace-write" {
		t.Errorf("the locked value must win, got %q", v.Value)
	}
	if !v.Locked {
		t.Error("the value should be marked locked")
	}
	if len(r.Warnings) == 0 {
		t.Fatal("the attempted override must be reported")
	}
	if !strings.Contains(r.Warnings[0], "DCODE_SANDBOX_MODE") {
		t.Errorf("the warning should name where the ignored value came from: %q", r.Warnings[0])
	}
}

// Provenance is what makes a configuration explainable.
func TestEveryValueCarriesItsOrigin(t *testing.T) {
	r := Resolve([]Layer{
		{Source: SourceUser, Origin: "~/.config/dcode/config.toml",
			Values: map[string]string{"model.name": "MiniMax-M3", "limits.max_iterations": "50"}},
	})
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		if v.Origin == "" {
			t.Errorf("%s has no origin; support cannot answer \"where did this come from\"", k)
		}
	}
}

func TestTypedAccessors(t *testing.T) {
	r := Resolve([]Layer{{Source: SourceUser, Origin: "f", Values: map[string]string{
		"s": "hello", "b_true": "true", "b_yes": "yes", "b_off": "off",
		"n": "42", "bad_n": "not-a-number", "empty": "",
	}}})

	if got := r.String("s", "def"); got != "hello" {
		t.Errorf("got %q", got)
	}
	if got := r.String("missing", "def"); got != "def" {
		t.Errorf("got %q", got)
	}
	if got := r.String("empty", "def"); got != "def" {
		t.Errorf("an empty value should fall back, got %q", got)
	}
	for k, want := range map[string]bool{"b_true": true, "b_yes": true, "b_off": false} {
		if got := r.Bool(k, !want); got != want {
			t.Errorf("%s: got %v want %v", k, got, want)
		}
	}
	if got := r.Bool("missing", true); !got {
		t.Error("a missing bool should fall back")
	}
	if got := r.Int("n", 0); got != 42 {
		t.Errorf("got %d", got)
	}
	// A malformed number must fall back rather than resolve to zero, which
	// would silently disable whatever it configures.
	if got := r.Int("bad_n", 7); got != 7 {
		t.Errorf("a malformed number should fall back, got %d", got)
	}
}

// ---------- roots ----------

func TestRootsAreSeparateByDefault(t *testing.T) {
	roots, err := DiscoverRoots(envFrom(map[string]string{"HOME": "/home/u"}))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" {
		if !strings.Contains(roots.Cache, "Caches") {
			t.Errorf("macOS cache root: %q", roots.Cache)
		}
		return
	}
	// Config belongs in a dotfiles repository; a session log does not. Sharing
	// one directory makes both impossible at once.
	if roots.Config == roots.State {
		t.Error("config and state must not share a directory")
	}
	if !strings.Contains(roots.Config, ".config") {
		t.Errorf("config root: %q", roots.Config)
	}
	if !strings.Contains(roots.Cache, ".cache") {
		t.Errorf("cache root: %q", roots.Cache)
	}
}

func TestXDGVariablesAreHonoured(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS uses Application Support unless XDG is set explicitly")
	}
	roots, err := DiscoverRoots(envFrom(map[string]string{
		"HOME": "/home/u", "XDG_CONFIG_HOME": "/custom/config",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if roots.Config != "/custom/config/dcode" {
		t.Errorf("got %q", roots.Config)
	}
}

func TestExplicitRootWins(t *testing.T) {
	roots, err := DiscoverRoots(envFrom(map[string]string{
		"HOME": "/home/u", "XDG_CONFIG_HOME": "/xdg", "DCODE_CONFIG_DIR": "/explicit",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if roots.Config != "/explicit" {
		t.Errorf("got %q", roots.Config)
	}
}

// The escape hatch has to collapse every root; one that escapes would create a
// directory outside the chosen home and be discovered by accident months later.
func TestDcodeHomeCollapsesEveryRoot(t *testing.T) {
	roots, err := DiscoverRoots(envFrom(map[string]string{
		"HOME": "/home/u", "DCODE_HOME": "/opt/dcode",
		"XDG_CONFIG_HOME": "/xdg", "XDG_CACHE_HOME": "/xdgcache",
	}))
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]string{
		"config": roots.Config, "data": roots.Data,
		"state": roots.State, "cache": roots.Cache,
	} {
		if !strings.HasPrefix(got, "/opt/dcode") {
			t.Errorf("%s root escaped DCODE_HOME: %q", name, got)
		}
	}
}

func TestDcodeHomeMustBeAbsolute(t *testing.T) {
	if _, err := DiscoverRoots(envFrom(map[string]string{"DCODE_HOME": "relative"})); err == nil {
		t.Error("a relative DCODE_HOME must be rejected")
	}
}

func TestMissingHomeIsAnError(t *testing.T) {
	if _, err := DiscoverRoots(envFrom(map[string]string{})); err == nil {
		t.Error("without HOME or DCODE_HOME there is nowhere to look")
	}
}

func TestEnsureCreatesOwnerOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dcode")
	if err := Ensure(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Config and state hold the user's instructions and history.
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("got %o want 700", perm)
	}
	if err := Ensure(""); err == nil {
		t.Error("an empty path must be rejected")
	}
}

// ---------- credential refusal ----------

// The check must cover unknown sections too: that is exactly where a secret
// would slip through unnoticed.
func TestCredentialsAreRefusedIncludingInUnknownSections(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  string
	}{
		{"api key", "[model]\napi_key = \"sk-x\"\n"},
		{"apikey", "[provider]\napikey = \"sk-x\"\n"},
		{"api-key", "[x]\napi-key = \"sk-x\"\n"},
		{"token", "[auth]\ntoken = \"t\"\n"},
		{"secret", "[whatever]\nsecret = \"s\"\n"},
		{"password", "[db]\npassword = \"p\"\n"},
		{"unknown section", "[totally-unknown]\napi_key = \"sk-x\"\n"},
		{"uppercase", "[model]\nAPI_KEY = \"sk-x\"\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Through the live parser rather than a private copy of the rule.
			// A test that exercises its own implementation of a guarantee
			// agrees with itself, which is the one thing a test must not do.
			_, err := ParseSections(tc.doc, "config.toml")
			if err == nil {
				t.Fatal("a credential-shaped key must be refused")
			}
			if !strings.Contains(err.Error(), "DCODE_API_KEY") {
				t.Errorf("the error must say where credentials should come from: %v", err)
			}
		})
	}
}

func TestOrdinaryKeysAreAccepted(t *testing.T) {
	_, err := ParseSections("[model]\nname = \"MiniMax-M3\"\n\n[sandbox]\nmode = \"workspace-write\"\n", "config.toml")
	if err != nil {
		t.Errorf("ordinary configuration must pass: %v", err)
	}
}

// ---------- instruction discovery ----------

func writeF(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoveryWalksRootDownToTheSessionDirectory(t *testing.T) {
	ws := t.TempDir()
	pkg := filepath.Join(ws, "packages", "api")

	writeF(t, ws, "AGENTS.md", "ROOT-RULE")
	writeF(t, filepath.Join(ws, "packages"), "AGENTS.md", "MID-RULE")
	writeF(t, pkg, "AGENTS.md", "LEAF-RULE")

	got, err := DiscoverInstructions(ws, pkg, nil, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 files, got %d: %+v", len(got), got)
	}
	order := []string{"ROOT-RULE", "MID-RULE", "LEAF-RULE"}
	for i, want := range order {
		if !strings.Contains(got[i].Text, want) {
			t.Errorf("position %d: want %s, got %q", i, want, got[i].Text)
		}
	}
	if got[0].Source != "project" {
		t.Errorf("the workspace root is the project scope, got %q", got[0].Source)
	}
	if got[2].Source != "directory" {
		t.Errorf("deeper levels are directory scope, got %q", got[2].Source)
	}
}

// In a nested monorepo, reading above the root would pull in a neighbouring
// project's rules and make behaviour inexplicable.
func TestDiscoveryNeverReadsAboveTheWorkspace(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "ws")
	writeF(t, root, "AGENTS.md", "OUTSIDE-RULE")
	writeF(t, ws, "AGENTS.md", "INSIDE-RULE")

	got, err := DiscoverInstructions(ws, ws, nil, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got {
		if strings.Contains(f.Text, "OUTSIDE-RULE") {
			t.Errorf("a file above the workspace was read: %s", f.Path)
		}
	}
	if len(got) != 1 {
		t.Errorf("want only the workspace file, got %d", len(got))
	}
}

// DCODE.md is more specific to this tool, so it comes last and therefore wins.
func TestDcodeFileWinsOverAgentsFileInTheSameDirectory(t *testing.T) {
	ws := t.TempDir()
	writeF(t, ws, "AGENTS.md", "SHARED")
	writeF(t, ws, "DCODE.md", "SPECIFIC")

	got, err := DiscoverInstructions(ws, ws, nil, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("both files should be read, got %d", len(got))
	}
	if !strings.Contains(got[0].Text, "SHARED") || !strings.Contains(got[1].Text, "SPECIFIC") {
		t.Errorf("DCODE.md must come last: %+v", got)
	}
}

// Truncating in silence leaves the user believing a rule is in force when it is
// not, which is worse than not reading the file at all.
func TestTruncationIsAnnouncedInTheText(t *testing.T) {
	ws := t.TempDir()
	writeF(t, ws, "AGENTS.md", strings.Repeat("x", 5000))

	got, err := DiscoverInstructions(ws, ws, nil, 100, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatal("expected one file")
	}
	if !strings.Contains(got[0].Text, "truncated") {
		t.Errorf("the truncation must be visible: %q", got[0].Text)
	}
}

func TestDepthLimitStopsTheWalk(t *testing.T) {
	ws := t.TempDir()
	deep := filepath.Join(ws, "a", "b", "c", "d")
	writeF(t, ws, "AGENTS.md", "ROOT")
	writeF(t, filepath.Join(ws, "a"), "AGENTS.md", "A")
	writeF(t, filepath.Join(ws, "a", "b"), "AGENTS.md", "B")
	writeF(t, deep, "AGENTS.md", "D")

	got, err := DiscoverInstructions(ws, deep, nil, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got {
		if strings.Contains(f.Text, "D") && !strings.Contains(f.Text, "ROOT") {
			t.Errorf("the depth limit was exceeded: %s", f.Path)
		}
	}
}

func TestDiscoveryWithNoFilesReturnsNothing(t *testing.T) {
	got, err := DiscoverInstructions(t.TempDir(), t.TempDir(), nil, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d", len(got))
	}
}

func TestCustomInstructionNames(t *testing.T) {
	ws := t.TempDir()
	writeF(t, ws, "CUSTOM.md", "CUSTOM-RULE")
	got, err := DiscoverInstructions(ws, ws, []string{"CUSTOM.md"}, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Text, "CUSTOM-RULE") {
		t.Errorf("got %+v", got)
	}
}

func TestDiscoveryOutsideWorkspaceFallsBackToTheRoot(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "ws")
	other := filepath.Join(root, "other")
	writeF(t, ws, "AGENTS.md", "WS-RULE")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := DiscoverInstructions(ws, other, nil, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Text, "WS-RULE") {
		t.Errorf("a directory outside the workspace should still get the root rules: %+v", got)
	}
}
