package app

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/loop"
)

// A KnownKeys entry that has a matching Options field but is not assigned by
// FromEnv, or is not passed through to loop.Config, is a config bug that
// survives every per-layer test: the schema accepts the key, the parser maps
// it, the struct has the slot, FromEnv just does not read it, and the user
// writes a setting they believe is in force.
//
// That is exactly what happened with behavior.show_reasoning: it existed in
// KnownKeys, existed as Options.ShowReasoning, was not assigned in FromEnv,
// and was not plumbed into loop.Config — the whole suite passed because no
// test crossed the three layers. The tests below are the cross-layer
// witnesses. Anything added to the wiring table without completing the
// chain in code must make them fail.

// configOption is one chain link: the Options field the chain ends at, the
// KnownKeys entry that drives it, the r accessor FromEnv uses, and the
// loop.Config field it must reach (empty when the value lives on Options
// and is consumed by app code, not the loop).
type configOption struct {
	opts     string // app.Options field name, e.g. "ShowReasoning"
	key      string // config.KnownKeys entry, e.g. "behavior.show_reasoning"
	resolver string // r accessor name: "Bool", "String" or "Int"
	loopCfg  string // loop.Config field name; "" when not plumbed to the loop
}

// wiringTable is the declaration every PR touching Options or loop.Config
// must keep in sync. Adding a row is the signal that the chain has been
// completed in both directions.
//
// When loopCfg is empty, the value lives on Options but is not consumed by
// the loop — a deliberate state, not a missed wiring.
var wiringTable = []configOption{
	{"Model", "model.name", "String", ""},
	{"Transport", "model.transport", "String", ""},
	{"Family", "model.family", "String", ""},
	{"BaseURL", "model.base_url", "String", ""},

	{"SandboxMode", "sandbox.mode", "String", "Mode"},
	{"Policy", "sandbox.approval_policy", "String", "Policy"},
	{"AllowNetwork", "sandbox.allow_network", "Bool", ""},
	{"Backend", "sandbox.backend", "String", ""},

	{"Parallel", "limits.parallel", "Int", "Parallel"},
	{"Limits", "limits.max_iterations", "Int", ""},
	{"Limits", "limits.identical", "Int", ""},
	{"Limits", "limits.max_turn_tokens", "Int", ""},

	{"Reminders", "behavior.reminders_enabled", "Bool", "Reminders"},
	{"ShowReasoning", "behavior.show_reasoning", "Bool", "ShowReasoning"},
	{"Instructions", "behavior.instructions_enabled", "Bool", ""},
	{"Skills", "behavior.skills_enabled", "Bool", ""},

	{"CredentialBackend", "credential.backend", "String", ""},

	{"Rules", "rules.confirm_write", "String", "Rules"},
	{"Rules", "rules.confirm_read", "String", "Rules"},
	{"Rules", "rules.confirm_command", "String", "Rules"},

	{"EditEchoDiff", "tools.edit_echo_diff", "String", ""},
	{"SymbolMaxMatches", "tools.symbol_max_matches", "Int", ""},
	{"BudgetNotice", "budget.notice", "Bool", "BudgetNotice"},

	{"VerifyCommand", "verify.command", "String", ""},
	{"InstructionNotice", "instruction.notice", "Bool", ""},
	{"InstructionForeign", "instruction.foreign", "String", ""},
	{"DoneTimeout", "done.timeout", "String", "DoneTimeout"},
	{"DoneEnabled", "done.enabled", "Bool", "DoneEnabled"},
	{"DoneFile", "done.file", "String", ""},
	{"MaxStallCycles", "limits.max_stall_cycles", "Int", "MaxStallCycles"},

	{"DumpPrompt", "doctrine.dump", "Bool", ""},
	{"DoctrineOverlay", "doctrine.enabled", "Bool", ""},
	{"DoctrineDir", "doctrine.dir", "String", ""},
	{"DoctrineMaxBytes", "doctrine.max_bytes", "Int", ""},
}

// nonSession names the KnownKeys entries that deliberately do not reach
// app.Options, with the reason each one does not.
//
// It is an escape hatch and it is deliberately narrow: an entry here is a
// promise that the key is read somewhere, and TestNonSessionKeysAreReadSomewhere
// makes that promise checkable. Without the second half this map would be the
// hole the whole file exists to close.
var nonSession = map[string]string{
	"update.check":   "read by the update path, which is not a session",
	"update.channel": "read by the update path, which is not a session",
	"eval.enabled":   "read by the eval harness, which measures the product rather than running it",
	"eval.model":     "read by the eval harness, which measures the product rather than running it",
	"eval.runs":      "read by the eval harness, which measures the product rather than running it",
}

// TestEveryKnownKeyIsAccountedFor is what the rest of this file was missing.
//
// The other legs all start from wiringTable, so a key added to KnownKeys and
// wired nowhere passed every one of them: no row, no assertion, no failure.
// That is precisely how limits.max_turn_tokens, behavior.instructions_enabled,
// behavior.skills_enabled and update.channel came to be declared, accepted,
// reported by `dcode config` with an origin — and read by nobody.
//
// Starting from KnownKeys instead of from the table inverts that. A new key
// fails until someone either wires it and declares the row, or states here why
// it does not belong to a session.
func TestEveryKnownKeyIsAccountedFor(t *testing.T) {
	inTable := map[string]bool{}
	for _, w := range wiringTable {
		inTable[w.key] = true
	}
	for key := range config.KnownKeys {
		if inTable[key] {
			continue
		}
		if _, ok := nonSession[key]; ok {
			continue
		}
		t.Errorf("KnownKeys declares %q, which no wiring row claims and "+
			"nonSession does not excuse: it is accepted by the schema and read "+
			"by nothing", key)
	}
	for key := range nonSession {
		if _, ok := config.KnownKeys[key]; !ok {
			t.Errorf("nonSession excuses %q, which is not a known key", key)
		}
		if inTable[key] {
			t.Errorf("%q is both wired and excused; one of the two is wrong", key)
		}
	}
}

// TestNonSessionKeysAreReadSomewhere is the other half of the escape hatch.
//
// A key excused from the wiring table still has to be read by something. The
// check looks for the key spelling in the sources of every consumer that is
// not a session, because that is the cheapest honest proof that the excuse is
// not just an excuse.
//
// This is the assertion that would have caught update.channel: the key mapped
// to DCODE_UPDATE_CHANNEL, the code read DCODE_RELEASE_CHANNEL, and nothing
// anywhere compared the two spellings.
func TestNonSessionKeysAreReadSomewhere(t *testing.T) {
	src := readNonSessionSources(t)
	for key := range nonSession {
		if !strings.Contains(src, `"`+key+`"`) {
			t.Errorf("%q is excused from the wiring table as %q, but no command "+
				"reads it by that name", key, nonSession[key])
		}
	}
}

// nonSessionDirs are the places a key excused from the wiring table may be
// consumed. Each one is a consumer that is not a session: the command layer,
// and the eval harness that measures the product rather than running it.
//
// Adding a directory here widens the escape hatch, so the list stays short and
// each entry names something that genuinely reads configuration.
var nonSessionDirs = [][]string{
	{"..", "..", "cmd", "dcode"},
	{"..", "evals"},
}

// readNonSessionSources concatenates the non-test Go sources of every consumer
// that is allowed to claim a non-session key.
func readNonSessionSources(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	for _, parts := range nonSessionDirs {
		dir := filepath.Join(parts...)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			b.Write(data)
		}
	}
	if b.Len() == 0 {
		t.Fatal("no consumer sources found; the guard would pass vacuously")
	}
	return b.String()
}

// TestWiringTablePointsAtRealKeys guards the table against typos in the key
// column. A row that names a key which is not in KnownKeys is dead
// documentation: the chain link it describes cannot exist.
func TestWiringTablePointsAtRealKeys(t *testing.T) {
	known := map[string]bool{}
	for k := range config.KnownKeys {
		known[k] = true
	}
	seen := map[string]bool{}
	for _, w := range wiringTable {
		if !known[w.key] {
			t.Errorf("wiring row points at %q, which is not in KnownKeys", w.key)
		}
		if seen[w.opts+"|"+w.key] {
			t.Errorf("wiring row %s/%s is declared twice", w.opts, w.key)
		}
		seen[w.opts+"|"+w.key] = true
	}
}

// TestOptionsHasEveryDeclaredField is the first leg: every Options field
// the table promises must exist and be exported.
func TestOptionsHasEveryDeclaredField(t *testing.T) {
	optsType := reflect.TypeOf(Options{})
	for _, w := range wiringTable {
		f, ok := optsType.FieldByName(w.opts)
		if !ok {
			t.Errorf("%s declares Options.%s, which does not exist", w.key, w.opts)
			continue
		}
		if !f.IsExported() {
			t.Errorf("%s declares Options.%s, which is unexported", w.key, w.opts)
		}
	}
}

// TestLoopConfigHasEveryDeclaredField is the second leg: every loop.Config
// field the table promises must exist and be exported.
func TestLoopConfigHasEveryDeclaredField(t *testing.T) {
	cfgType := reflect.TypeOf(loop.Config{})
	for _, w := range wiringTable {
		if w.loopCfg == "" {
			continue
		}
		f, ok := cfgType.FieldByName(w.loopCfg)
		if !ok {
			t.Errorf("%s declares loop.Config.%s, which does not exist",
				w.key, w.loopCfg)
			continue
		}
		if !f.IsExported() {
			t.Errorf("%s declares loop.Config.%s, which is unexported",
				w.key, w.loopCfg)
		}
	}
}

// TestFromEnvPopulatesEveryDeclaredField is the third leg: FromEnv must
// assign every Options field declared in the table, and must do so via a
// Resolved accessor for the declared key.
//
// The assertion inspects the source text rather than the values. The reason
// is the bug we are guarding against: FromEnv can be syntactically complete
// and still skip a field — its zero value is just as good as any other
// until the environment overrides it, and a runtime test would pass.
//
// This is the exact shape of the show_reasoning bug: the field existed, the
// call to FromEnv ran without error, and the field stayed at the zero
// value because no line in FromEnv touched it.
//
// The check accepts both `=` and `:` after the field name so a struct
// literal like `Model: r.String(...)` counts as an assignment.
//
// The accessor check accepts the call as long as it appears in FromEnv's
// own body, or — for fields that go through a helper (resolveRules is the
// only one today) — as long as FromEnv wires the helper. We track which
// fields go through a helper in helperKeys; a row that names a helper key
// must still appear in the FromEnv body via the helper call.
func TestFromEnvPopulatesEveryDeclaredField(t *testing.T) {
	body := readFromEnvBody(t)
	for _, w := range wiringTable {
		assigned := strings.Contains(body, w.opts+"=") ||
			strings.Contains(body, w.opts+":")
		if !assigned {
			t.Errorf("FromEnv never assigns Options.%s (chain link for %s)",
				w.opts, w.key)
		}
		if helperKeys[w.key] {
			// FromEnv does not read it directly; resolveRules does.
			// The wiring is still asserted because the resolver call sits
			// in FromEnv's body — that is the structural proof.
			if !strings.Contains(body, "resolveRules(r)") &&
				!strings.Contains(body, "resolveRules(r,") {
				t.Errorf("FromEnv references resolveRules, but not via a helper call")
			}
			continue
		}
		accessor := w.resolver + `("` + w.key + `"`
		if !strings.Contains(body, accessor) {
			t.Errorf("FromEnv never reads %s via r.%s for Options.%s",
				w.key, w.resolver, w.opts)
		}
	}
}

// helperKeys names KnownKeys whose accessor call lives in a helper that
// FromEnv calls. Today that is the three rules.confirm_* entries, which
// FromEnv routes through resolveRules. Adding a new helper means adding a
// new entry here, with the wiring call FromEnv makes for it.
var helperKeys = map[string]bool{
	"rules.confirm_write":   true,
	"rules.confirm_read":    true,
	"rules.confirm_command": true,
}

// TestNewWiresEveryLoopConfigField is the fourth leg: when a row names a
// loop.Config field, the call to loop.New inside app.New must pass it from
// opts. This is the other half of the show_reasoning bug: the loop field
// existed, the Options field existed, and the line that copied one into
// the other was absent.
//
// The match is narrow on purpose: a wrapper that still passes the value
// through is fine, but the line must exist and must reference
// opts.<Field>. That is what makes a missing assignment a failure, not a
// passing test with the value at zero.
func TestNewWiresEveryLoopConfigField(t *testing.T) {
	body := readAppNewBody(t)
	for _, w := range wiringTable {
		if w.loopCfg == "" {
			continue
		}
		marker := w.loopCfg + ":"
		idx := strings.Index(body, marker)
		if idx < 0 {
			t.Errorf("app.New does not build a loop.Config with %s set",
				w.loopCfg)
			continue
		}
		rest := body[idx+len(marker):]
		end := strings.IndexAny(rest, ",}")
		if end < 0 {
			end = len(rest)
		}
		value := rest[:end]
		if !strings.Contains(value, "opts."+w.opts) {
			t.Errorf("loop.Config.%s is set, but not from opts.%s (chain link for %s)",
				w.loopCfg, w.opts, w.key)
		}
	}
}

// TestFromEnvOverridesReachTheLoop is the fifth and last leg: the runtime
// proof that the chain is not just declared but live. Setting an
// environment variable changes the value that arrives at loop.Config.
//
// This is what the show_reasoning bug did not have: no test asserted that
// the env var actually changed the field on the engine, so the missing
// assignment went unnoticed.
//
// Sandbox.New is platform-dependent; on a host without a sandbox backend
// the test is skipped, which is the honest outcome — the structural legs
// above are what this bug class is really about, and the runtime check is
// the bonus that turns "the line looks right" into "the value arrives".
func TestFromEnvOverridesReachTheLoop(t *testing.T) {
	cases := []struct {
		name      string
		envVar    string
		set, want string
		// "bool", "string" or "int" — picks the right reflect comparison
		// for both ends of the chain. A row that mixes them would be a
		// config bug of its own, and the test surfaces it directly.
		kind      string
		optsField string
		loopField string
	}{
		// Behaviour keys must reach the loop. This is the exact regression
		// shape: the key, the field, and the plumbed value.
		{"reasoning off", "DCODE_SHOW_REASONING", "false", "false",
			"bool", "ShowReasoning", "ShowReasoning"},
		{"reminders off", "DCODE_BEHAVIOR_REMINDERS_ENABLED", "false", "false",
			"bool", "Reminders", "Reminders"},

		// Sandbox knobs that have a matching loop.Config field.
		{"sandbox mode", "DCODE_SANDBOX_MODE", "read-only", "read-only",
			"string", "SandboxMode", "Mode"},
		{"approval policy", "DCODE_APPROVAL_POLICY", "never", "never",
			"string", "Policy", "Policy"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts, _, err := FromEnv(envFrom(map[string]string{c.envVar: c.set}), t.TempDir())
			if err != nil {
				t.Fatalf("FromEnv: %v", err)
			}

			// Reach the field through a pointer so the reflect.Value is
			// addressable. A value returned from a function is on the stack
			// and not addressable; UnsafeAddr panics on those.
			optsCopy := opts
			gotOpts := reflect.Indirect(reflect.ValueOf(&optsCopy)).FieldByName(c.optsField)
			if !gotOpts.IsValid() {
				t.Fatalf("Options has no field %s", c.optsField)
			}
			assertKind(t, "Options."+c.optsField, gotOpts, c.kind, c.want)

			cfg := loopConfigFromOptions(t, opts)
			gotLoop := cfg.FieldByName(c.loopField)
			if !gotLoop.IsValid() {
				t.Fatalf("loop.Config has no field %s", c.loopField)
			}
			assertKind(t, "loop.Config."+c.loopField, gotLoop, c.kind, c.want)
		})
	}
}

// assertKind compares a reflect.Value to the want string under the kind's
// own equality. A bool and a string are not interchangeable, and a test
// that confuses them would silently accept a missing wiring.
//
// Values reached through an unexported field cannot be obtained with
// reflect.Value.Interface, so we read them via unsafe.Pointer. The pointer
// trick is what makes the runtime check possible at all: the loop keeps
// its config unexported by design, and reflection is the only honest way to
// inspect it from outside the package.
func assertKind(t *testing.T, label string, v reflect.Value, kind, want string) {
	t.Helper()
	switch kind {
	case "bool":
		got := readBool(v)
		var parsed bool
		switch want {
		case "true":
			parsed = true
		case "false":
			parsed = false
		default:
			t.Fatalf("bool row has want=%q, which is neither true nor false", want)
		}
		if got != parsed {
			t.Errorf("%s = %v, want %v", label, got, parsed)
		}
	case "string":
		if got := readString(v); got != want {
			t.Errorf("%s = %q, want %q", label, got, want)
		}
	case "int":
		var wantInt int
		if _, err := fmt.Sscanf(want, "%d", &wantInt); err != nil {
			t.Fatalf("int row has want=%q, not an integer: %v", want, err)
		}
		if v.Int() != int64(wantInt) {
			t.Errorf("%s = %d, want %d", label, v.Int(), wantInt)
		}
	default:
		t.Fatalf("unknown kind %q", kind)
	}
}

// readBool reads a bool even from an unexported field. reflect.Value.Bool
// itself is fine; only Interface() panics. We only need a bool out of it.
func readBool(v reflect.Value) bool {
	return *(*bool)(unsafe.Pointer(v.UnsafeAddr()))
}

// readString reads a string even from an unexported field. String headers
// carry a pointer and a length; copying the header out of the unsafe
// pointer gives a Go string that points into the same backing memory. The
// test does not mutate it, so the aliasing is safe.
func readString(v reflect.Value) string {
	if v.Kind() != reflect.String {
		return ""
	}
	return *(*string)(unsafe.Pointer(v.UnsafeAddr()))
}

// ---------- helpers ----------

// readFromEnvBody returns the body of FromEnv as a string, extracted from
// app.go by balancing braces from the function header.
// The bodies are concatenated because the chain is allowed to be split across
// a helper that FromEnv calls, and it is: FromEnv resolves the layers and
// fromResolved turns them into Options. Naming both here rather than only
// FromEnv is what keeps a behaviour-preserving refactor from failing a test
// about wiring — and it is the cost of asserting on source text at all.
//
// A new helper in the chain has to be named here. That is not overhead: an
// assignment reachable from nowhere in this list is an assignment no leg of
// this file can see.
func readFromEnvBody(t *testing.T) string {
	t.Helper()
	src := readAppSource(t)
	return extractFunctionBody(src, "FromEnv") +
		"\n" + extractFunctionBody(src, "fromResolved")
}

// readAppNewBody returns the body of New as a string.
func readAppNewBody(t *testing.T) string {
	t.Helper()
	return extractFunctionBody(readAppSource(t), "New")
}

// loopConfigFromOptions builds a session through the public wiring and
// returns the loop.Config value via reflection. The engine keeps its
// config unexported; reflection is the only way to reach it from outside
// the package, and it is the right tool here — the test is asserting a
// structural fact about the wiring, not running a turn.
//
// The returned value is addressable: we hold it through a pointer to a
// local copy, so subsequent UnsafeAddr calls in assertKind do not panic.
func loopConfigFromOptions(t *testing.T, opts Options) reflect.Value {
	t.Helper()
	sess, err := New(opts, nil, DenyAll{})
	if err != nil {
		// Sandbox is platform-dependent; a missing sandbox is not the
		// bug this test guards against.
		t.Skipf("New: %v", err)
	}
	enginePtr := sess.Engine
	engineVal := reflect.ValueOf(enginePtr).Elem()
	cfg := engineVal.FieldByName("cfg")
	// Take a pointer to the cfg value so it is addressable, then
	// indirect back to a value. This dance is what makes the unsafe read
	// in assertKind safe.
	cfgAddr := reflect.ValueOf(unsafe.Pointer(cfg.UnsafeAddr()))
	return reflect.NewAt(cfg.Type(), cfgAddr.UnsafePointer()).Elem()
}

// readAppSource loads app.go from the same directory the test lives in.
func readAppSource(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatalf("read app.go: %v", err)
	}
	return string(src)
}

// extractFunctionBody returns the body of `func name(` from src, between
// the opening `{` after the parameter list and the matching closing `}`.
func extractFunctionBody(src, name string) string {
	open := "func " + name + "("
	i := strings.Index(src, open)
	if i < 0 {
		return ""
	}
	depth := 0
	for j := i + len(open); j < len(src); j++ {
		switch src[j] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				k := strings.Index(src[j:], "{")
				if k < 0 {
					return ""
				}
				start := j + k + 1
				return extractBalanced(src, start)
			}
			depth--
		}
	}
	return ""
}

func extractBalanced(src string, start int) string {
	depth := 1
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start:i]
			}
		}
	}
	return src[start:]
}
