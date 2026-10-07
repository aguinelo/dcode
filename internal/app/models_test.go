package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/credential"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
)

// modelsKey is long and unlike anything else a response holds, so finding any
// part of it in one is finding the key.
const modelsKey = "sk-models-route-key-0123456789abcdef"

// modelsDaemon is a daemon booted in a directory of its own and resolved from a
// config root of its own: userModels as its models.toml, and a key in the
// environment, so every choice it lists has a credential behind it.
func modelsDaemon(t *testing.T, userModels string) (d *Daemon, booted string) {
	t.Helper()
	home := t.TempDir()
	if userModels != "" {
		write(t, home, "models.toml", userModels)
	}
	booted = t.TempDir()
	base, _, err := FromEnv(envFrom(map[string]string{
		"DCODE_HOME": home, "DCODE_API_KEY": modelsKey,
	}), booted)
	if err != nil {
		t.Fatal(err)
	}
	return NewDaemon(DaemonOptions{SocketPath: filepath.Join(t.TempDir(), "d.sock"), Base: base}), booted
}

// askModels asks through the daemon's own routes, as a client does, and hands
// back the raw body beside the decoded one: what must never be in a response is
// looked for in the bytes, not in a struct that has no field to hold it.
func askModels(t *testing.T, d *Daemon, workspace string) (int, string, protocol.ModelsResponse) {
	t.Helper()
	path := "/" + protocol.Version + "/models"
	if workspace != "" {
		path += "?workspace=" + url.QueryEscape(workspace)
	}
	rec := httptest.NewRecorder()
	d.server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var out protocol.ModelsResponse
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("%v: %s", err, rec.Body.String())
		}
	}
	return rec.Code, rec.Body.String(), out
}

// measuredAs is what a choice of family says about measurement: nothing, or
// the family's own admission. Asked of provider rather than typed here, so a
// family measured tomorrow does not turn this test into a claim about today.
func measuredAs(c protocol.ModelChoice) protocol.ModelChoice {
	c.Notice = provider.Unmeasured(c.Family)
	c.Measured = c.Notice == ""
	return c
}

// The menu of a workspace is what a session there can ask for: the model it
// gets when it asks for none, from that project's configuration, and every
// profile it can name — the user's, with the project's layered over them by
// name — sorted by name, each resolved as the session asking for it resolves
// it. Without a workspace, the daemon answers for the configuration it started
// with.
func TestTheMenuListsTheDefaultAndTheProfilesOfTheWorkspace(t *testing.T) {
	d, booted := modelsDaemon(t, "[profile.cloud]\nmodel = \"MiniMax-M3\"\n\n"+
		"[profile.sonnet]\nmodel = \"claude-sonnet-4\"\n")
	ws := t.TempDir()
	projectConfig(t, ws, "[model]\nname = \"gemini-2.5-pro\"\n")
	write(t, ws, filepath.Join(".dcode", "models.toml"),
		"[profile.cloud]\nmodel = \"gemini-2.5-flash\"\n"+
			"base_url = \"https://generativelanguage.googleapis.com/v1beta/openai\"\n\n"+
			"[profile.local]\nmodel = \"qwen3.5-9b\"\nfamily = \"generic\"\n"+
			"base_url = \"http://127.0.0.1:1234/v1\"\nwindow = 32000\n")

	code, body, got := askModels(t, d, ws)
	if code != http.StatusOK {
		t.Fatalf("status %d: %s", code, body)
	}
	if want := measuredAs(protocol.ModelChoice{Name: "gemini-2.5-pro", Model: "gemini-2.5-pro",
		Family: "gemini", Transport: "openai", Window: 1_000_000}); got.Default != want {
		t.Errorf("the default is\n  %+v\nwant the project's own\n  %+v", got.Default, want)
	}
	want := []protocol.ModelChoice{
		measuredAs(protocol.ModelChoice{Name: "cloud", Model: "gemini-2.5-flash", Family: "gemini",
			Transport: "openai", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
			Window: 1_000_000}),
		measuredAs(protocol.ModelChoice{Name: "local", Model: "qwen3.5-9b", Family: "generic",
			Transport: "openai", BaseURL: "http://127.0.0.1:1234/v1", Window: 32000}),
		measuredAs(protocol.ModelChoice{Name: "sonnet", Model: "claude-sonnet-4", Family: "claude",
			Transport: "anthropic", Window: 200_000}),
	}
	if !reflect.DeepEqual(got.Profiles, want) {
		t.Errorf("the profiles are\n  %+v\nwant\n  %+v", got.Profiles, want)
	}

	// Without a workspace: the configuration the daemon started with, which has
	// the user's profiles and none of that project's.
	code, body, own := askModels(t, d, "")
	if code != http.StatusOK {
		t.Fatalf("status %d without a workspace: %s", code, body)
	}
	if own.Default.Name != "MiniMax-M3" || len(own.Profiles) != 2 ||
		own.Profiles[0].Name != "cloud" || own.Profiles[0].Model != "MiniMax-M3" ||
		own.Profiles[1].Name != "sonnet" {
		t.Errorf("the daemon's own menu is %+v", own)
	}
	if _, _, again := askModels(t, d, booted); !reflect.DeepEqual(again, own) {
		t.Errorf("the directory the daemon started in answers\n  %+v\nand no workspace\n  %+v", again, own)
	}
}

// A model whose family nobody measured says so before anyone chooses it: not
// measured, with the family's own admission as its notice. A measured one says
// nothing, because a warning on everything is a warning nobody reads.
func TestAModelWithNoMeasurementSaysSo(t *testing.T) {
	d, booted := modelsDaemon(t, "[profile.local]\nmodel = \"qwen3.5-9b\"\nfamily = \"generic\"\n"+
		"base_url = \"http://127.0.0.1:1234/v1\"\n")
	code, body, got := askModels(t, d, booted)
	if code != http.StatusOK {
		t.Fatalf("status %d: %s", code, body)
	}
	if len(got.Profiles) != 1 {
		t.Fatalf("the profiles are %+v", got.Profiles)
	}
	if local := got.Profiles[0]; local.Measured || local.Notice != provider.GenericWarning {
		t.Errorf("a generic model reads as %+v, want not measured, with the generic warning", local)
	}
	if !strings.Contains(body, `"measured":false`) {
		t.Errorf("not measured is left for the client to infer from a missing field: %s", body)
	}
	if def := got.Default; !def.Measured || def.Notice != "" || def.Family != "minimax-m3" {
		t.Errorf("the measured default reads as %+v, want measured and silent", def)
	}
}

// Never a credential. Every choice has the key behind it — a session asking for
// any of them uses it — and the list carries none of it: not the key, not its
// mask, not its fingerprint, not where it was found. The fields are a closed
// set too, so a field added later is a decision made here rather than a leak.
func TestTheListOfModelsNeverCarriesACredential(t *testing.T) {
	d, booted := modelsDaemon(t, "[profile.local]\nmodel = \"qwen3.5-9b\"\nfamily = \"generic\"\n")
	if d.opts.Base.APIKey != modelsKey {
		t.Fatal("the daemon holds no key, and this test would pass by having nothing to leak")
	}
	secrets := map[string]string{
		"the key":            modelsKey,
		"its mask":           credential.Mask(modelsKey),
		"its fingerprint":    credential.Fingerprint(modelsKey),
		"where it was found": d.opts.Base.CredentialFrom,
	}
	fields := map[string]bool{"name": true, "model": true, "family": true, "transport": true,
		"base_url": true, "window": true, "measured": true, "notice": true}

	for _, ws := range []string{"", booted} {
		code, body, _ := askModels(t, d, ws)
		if code != http.StatusOK {
			t.Fatalf("status %d: %s", code, body)
		}
		for what, secret := range secrets {
			if secret == "" {
				t.Fatalf("%s is empty, so looking for it would find nothing", what)
			}
			if strings.Contains(body, secret) {
				t.Errorf("the list carries %s (%q): %s", what, secret, body)
			}
		}

		var top map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &top); err != nil {
			t.Fatal(err)
		}
		for field := range top {
			if field != "default" && field != "profiles" {
				t.Errorf("the answer carries %q, which this route does not declare: %s", field, body)
			}
		}
		var raw struct {
			Default  map[string]any   `json:"default"`
			Profiles []map[string]any `json:"profiles"`
		}
		if err := json.Unmarshal([]byte(body), &raw); err != nil {
			t.Fatal(err)
		}
		for _, choice := range append([]map[string]any{raw.Default}, raw.Profiles...) {
			for field := range choice {
				if !fields[field] {
					t.Errorf("a choice carries %q, which this route does not declare: %s", field, body)
				}
			}
		}
	}
}

// A workspace that cannot be one is refused, as a session there would be: a
// relative path before anything is read, a project whose configuration cannot
// be read with what is wrong in it. Never answered with the daemon's own
// configuration in its place, which would offer models the project never chose.
func TestAWorkspaceThatCannotBeReadIsRefusedWithTheReason(t *testing.T) {
	d, _ := modelsDaemon(t, "")

	brokenModels := t.TempDir()
	write(t, brokenModels, filepath.Join(".dcode", "models.toml"), "[profile.x]\nfamily = \"generic\"\n")
	brokenConfig := t.TempDir()
	projectConfig(t, brokenConfig, "[model]\nnmae = \"claude-opus-4\"\n")

	for _, tc := range []struct{ ws, reason string }{
		{"projects/dcode", "absolute"},
		{filepath.Join(t.TempDir(), "gone"), "cannot be opened"},
		{brokenModels, "has no model"},
		{brokenConfig, "nmae"},
	} {
		code, body, _ := askModels(t, d, tc.ws)
		if code != http.StatusBadRequest || !strings.Contains(body, protocol.CodeWorkspaceInvalid) ||
			!strings.Contains(body, tc.reason) {
			t.Errorf("%s: got %d %s, want 400 %s saying %q",
				tc.ws, code, body, protocol.CodeWorkspaceInvalid, tc.reason)
		}
	}
}

// A profile no session could be built with stays on the menu, not measured,
// and says why in the words the session would refuse with. Dropped, it would
// leave a person looking for the profile they wrote; called measured, it would
// claim measurements for a family nothing here can resolve.
func TestAProfileNoSessionCanBuildIsListedWithTheReason(t *testing.T) {
	d, booted := modelsDaemon(t, "[profile.llama]\nmodel = \"llama3\"\n\n"+
		"[profile.typo]\nmodel = \"MiniMax-M3\"\nfamily = \"minimax\"\n")
	code, body, got := askModels(t, d, booted)
	if code != http.StatusOK {
		t.Fatalf("a profile no session can build took the whole menu down: %d %s", code, body)
	}
	byName := map[string]protocol.ModelChoice{}
	for _, c := range got.Profiles {
		byName[c.Name] = c
	}
	for name, reason := range map[string]string{
		"llama": `no family claims model "llama3"`,
		"typo":  `no family named "minimax"`,
	} {
		c, ok := byName[name]
		if !ok {
			t.Errorf("%s was dropped from the menu: %+v", name, got.Profiles)
			continue
		}
		if c.Measured || !strings.Contains(c.Notice, reason) {
			t.Errorf("%s reads as %+v, want not measured and saying %q", name, c, reason)
		}
	}
}

// A choice is the session it opens: asking for each name builds a session with
// the model and the window the menu promised. It is what keeps the menu and the
// session resolving by one path, rather than by two that agree today.
func TestAChoiceDescribesTheSessionItOpens(t *testing.T) {
	d, booted := modelsDaemon(t, "[profile.local]\nmodel = \"qwen3.5-9b\"\nfamily = \"generic\"\nwindow = 32000\n\n"+
		"[profile.sonnet]\nmodel = \"claude-sonnet-4\"\n")
	requireSandbox(t, d.opts.Base)
	code, body, got := askModels(t, d, booted)
	if code != http.StatusOK {
		t.Fatalf("status %d: %s", code, body)
	}
	// Asking for no model is asking for the default.
	asked := map[string]protocol.ModelChoice{"": got.Default}
	for _, c := range got.Profiles {
		asked[c.Name] = c
	}
	for name, choice := range asked {
		sess, err := d.build(protocol.CreateSessionRequest{Workspace: booted, Model: name})
		if err != nil {
			t.Fatalf("asking for %q: %v", name, err)
		}
		opened := sess.Describe()
		sess.Close()
		if opened.Model != choice.Model || opened.ContextWindow != choice.Window {
			t.Errorf("asking for %q opened %s with a window of %d; the menu said %s and %d",
				name, opened.Model, opened.ContextWindow, choice.Model, choice.Window)
		}
	}
}
