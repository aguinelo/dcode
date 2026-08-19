package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// KnownKeys is the whole configuration surface, mapping each `section.key` to
// the one environment variable that carries it.
//
// DCODE_REQUIREMENTS_FILE is deliberately NOT here, and the omission is the
// point. Every key in this map can be set from config.toml — a file the user
// owns — and letting that file say where the locked policy lives would let the
// user redirect the policy that binds them. It is environment-only, like the
// directory roots, and for the same class of reason.
//
// The map is the schema. An unknown key is an error rather than a warning
// because a typo that is silently ignored is the most frustrating class of
// configuration bug there is: everything reports success and nothing changed.
//
// The mapping is bijective by contract — one key, one variable, in both
// directions — which is what lets `dcode --config <key>` name a single origin.
var KnownKeys = map[string]string{
	"model.name":                    "DCODE_MODEL",
	"model.transport":               "DCODE_TRANSPORT",
	"model.family":                  "DCODE_FAMILY",
	"model.base_url":                "DCODE_BASE_URL",
	"sandbox.mode":                  "DCODE_SANDBOX_MODE",
	"sandbox.approval_policy":       "DCODE_APPROVAL_POLICY",
	"sandbox.allow_network":         "DCODE_ALLOW_NETWORK",
	"sandbox.backend":               "DCODE_SANDBOX_BACKEND",
	"limits.max_iterations":         "DCODE_MAX_ITERATIONS",
	"limits.max_turn_tokens":        "DCODE_MAX_TURN_TOKENS",
	"limits.identical":              "DCODE_MAX_IDENTICAL_CALLS",
	"limits.parallel":               "DCODE_TOOL_PARALLELISM",
	"behavior.instructions_enabled": "DCODE_BEHAVIOR_INSTRUCTIONS_ENABLED",
	"behavior.skills_enabled":       "DCODE_BEHAVIOR_SKILLS_ENABLED",
	"behavior.reminders_enabled":    "DCODE_BEHAVIOR_REMINDERS_ENABLED",
	"behavior.show_reasoning":       "DCODE_SHOW_REASONING",
	"credential.backend":            "DCODE_CREDENTIAL_BACKEND",
	"rules.confirm_write":           "DCODE_CONFIRM_WRITE",
	"rules.confirm_read":            "DCODE_CONFIRM_READ",
	"rules.confirm_command":         "DCODE_CONFIRM_COMMAND",
	"update.check":                  "DCODE_UPDATE_CHECK",
	"update.channel":                "DCODE_UPDATE_CHANNEL", // DCODE_RELEASE_CHANNEL still works as a fallback
	"tools.edit_echo_diff":          "DCODE_EDIT_ECHO_DIFF",
	"tools.symbol_max_matches":      "DCODE_SYMBOL_MAX_MATCHES",
	"sandbox.unreadable":            "DCODE_SANDBOX_UNREADABLE",
	"sandbox.sockets":               "DCODE_SANDBOX_SOCKETS",
	"sandbox.writable":              "DCODE_SANDBOX_WRITABLE",
	"tools.fetch_enabled":           "DCODE_FETCH_ENABLED",
	"tools.fetch_max_bytes":         "DCODE_FETCH_MAX_BYTES",
	"delegate.enabled":              "DCODE_DELEGATE_ENABLED",
	"delegate.max_iterations":       "DCODE_DELEGATE_MAX_ITERATIONS",
	"delegate.max_result_bytes":     "DCODE_DELEGATE_MAX_RESULT_BYTES",
	"budget.notice":                 "DCODE_BUDGET_NOTICE",
	"verify.command":                "DCODE_VERIFY_COMMAND",
	"instruction.notice":            "DCODE_INSTRUCTION_NOTICE",
	"instruction.foreign":           "DCODE_INSTRUCTION_FOREIGN",
	"done.file":                     "DCODE_DONE_FILE",
	"done.enabled":                  "DCODE_DONE_ENABLED",
	"done.timeout":                  "DCODE_DONE_TIMEOUT",
	"limits.max_stall_cycles":       "DCODE_MAX_STALL_CYCLES",
	"ui.lang":                       "DCODE_LANG",
	"record.enabled":                "DCODE_RECORD_ENABLED",
	"record.dir":                    "DCODE_RECORD_DIR",
	"record.keep_days":              "DCODE_RECORD_KEEP_DAYS",
	"record.max_bytes":              "DCODE_RECORD_MAX_BYTES",
	"doctrine.dump":                 "DCODE_DOCTRINE_DUMP",
	"doctrine.enabled":              "DCODE_DOCTRINE_ENABLED",
	"doctrine.dir":                  "DCODE_DOCTRINE_DIR",
	"doctrine.max_bytes":            "DCODE_DOCTRINE_MAX_BYTES",
	// The eval keys are read by the measurement harness, never by the product.
	// They live in the same schema anyway: a key that governs behaviour and
	// cannot be inspected with `--config` is the gap the audit pair closes,
	// and "behaviour" here includes what the thresholds were measured against.
	"eval.enabled": "DCODE_EVAL_ENABLED",
	"eval.model":   "DCODE_EVAL_MODEL",
	"eval.runs":    "DCODE_EVAL_RUNS",
}

// EnvToKey inverts KnownKeys. Built once so the bijection can be asserted.
var EnvToKey = func() map[string]string {
	m := make(map[string]string, len(KnownKeys))
	for k, v := range KnownKeys {
		m[v] = k
	}
	return m
}()

// ConfigFileName is the file every root looks for.
const ConfigFileName = "config.toml"

// ParseTOML reads the configuration subset of TOML: sections, and keys whose
// values are strings, booleans or integers.
//
// A full TOML parser is a dependency and a much larger surface than the file
// documented in the spec. Every construct outside that subset is rejected by
// name rather than ignored, so a user writing valid TOML that dcode does not
// support is told so instead of watching it do nothing.
func ParseTOML(data []byte, origin string) (map[string]string, error) {
	values := map[string]string{}
	section := ""

	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		where := fmt.Sprintf("%s:%d", origin, i+1)

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, fmt.Errorf("%s: unterminated section header", where)
			}
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section == "" {
				return nil, fmt.Errorf("%s: empty section name", where)
			}
			if strings.HasPrefix(section, "[") {
				return nil, fmt.Errorf("%s: arrays of tables are not supported", where)
			}
			continue
		}

		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return nil, fmt.Errorf("%s: expected `key = value`", where)
		}
		name := strings.TrimSpace(line[:eq])
		val, err := parseValue(strings.TrimSpace(line[eq+1:]))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", where, err)
		}
		if name == "" {
			return nil, fmt.Errorf("%s: empty key", where)
		}
		if section == "" {
			return nil, fmt.Errorf("%s: key %q sits outside any section", where, name)
		}

		key := section + "." + name
		// The credential check runs before the unknown-key check, and on the
		// key's own name rather than the resolved one, so a secret parked in a
		// section dcode does not know about is still refused (RN-3).
		if credentialKey.MatchString(name) {
			return nil, credentialError(key, origin)
		}
		if _, ok := KnownKeys[key]; !ok {
			return nil, fmt.Errorf("%s: unknown key %q.\nKnown keys:\n%s",
				where, key, knownKeyList())
		}
		values[key] = val
	}
	return values, nil
}

// stripComment removes a trailing `#` comment, honouring quotes so a `#` inside
// a string value survives.
func stripComment(s string) string {
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				return s[:i]
			}
		}
	}
	return s
}

func parseValue(s string) (string, error) {
	switch {
	case s == "":
		return "", fmt.Errorf("missing value")
	case strings.HasPrefix(s, `"`):
		if len(s) < 2 || !strings.HasSuffix(s, `"`) {
			return "", fmt.Errorf("unterminated string")
		}
		return s[1 : len(s)-1], nil
	case s == "true" || s == "false":
		return s, nil
	case strings.HasPrefix(s, "["):
		// A list of patterns is genuinely a list, so the parser accepts one and
		// carries it as a single value — the key-to-variable mapping stays
		// bijective, which is what lets `--config` name one origin.
		return parseStringArray(s)
	case strings.HasPrefix(s, "{"):
		return "", fmt.Errorf("inline tables are not supported")
	default:
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return "", fmt.Errorf("value %q is not a string, a boolean or a number", s)
		}
		return s, nil
	}
}

// parseStringArray reads `["a", "b"]` into `a,b`.
func parseStringArray(s string) (string, error) {
	if !strings.HasSuffix(s, "]") {
		return "", fmt.Errorf("unterminated array")
	}
	body := strings.TrimSpace(s[1 : len(s)-1])
	if body == "" {
		return "", nil
	}
	var out []string
	for _, part := range strings.Split(body, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.HasPrefix(part, `"`) || !strings.HasSuffix(part, `"`) || len(part) < 2 {
			return "", fmt.Errorf("an array may only hold quoted strings, got %s", part)
		}
		item := part[1 : len(part)-1]
		if strings.Contains(item, ",") {
			// The separator is what joins them back, so an item carrying one
			// would come out as two.
			return "", fmt.Errorf("a list item cannot contain a comma: %q", item)
		}
		out = append(out, item)
	}
	return strings.Join(out, ","), nil
}

func knownKeyList() string {
	keys := make([]string, 0, len(KnownKeys))
	for k := range KnownKeys {
		keys = append(keys, "  "+k)
	}
	sort.Strings(keys)
	return strings.Join(keys, "\n")
}

// LoadFile reads a config.toml into a layer.
//
// A missing file is not an error: configuration is optional at every level, and
// the default layer already answers every key.
func LoadFile(path string, source Source) (Layer, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Layer{}, false, nil
	}
	if err != nil {
		return Layer{}, false, err
	}
	values, err := ParseTOML(data, path)
	if err != nil {
		return Layer{}, false, err
	}
	return Layer{Source: source, Origin: path, Values: values}, true, nil
}

// FileLayers loads the user and project configuration files, in the order the
// precedence chain expects them.
// RequirementsFileName is the administrator's locked configuration.
//
// A separate file from config.toml, and that separation is the whole point: it
// is what an organisation deploys and what a user cannot edit, so mixing the
// two into one file with a "locked" marker per key would make the boundary a
// convention inside a file the user owns.
const RequirementsFileName = "requirements.toml"

// FileLayers loads every configuration file, weakest first.
//
// The locked layer is loaded LAST and ranked HIGHEST, and that is RN-7 of
// sandbox-policy and RN-9 of configuration: an administrator's policy is not
// overridable by an environment variable or by a flag. It is what makes dcode
// adoptable in an organisation, and it was the one layer nothing ever built —
// SourceLocked was defined, ranked, and handled by Resolve, and no production
// code path constructed it.
func FileLayers(roots Roots, workspace, requirements string) ([]Layer, error) {
	var out []Layer
	sources := []struct {
		path   string
		source Source
	}{
		{filepath.Join(roots.Config, ConfigFileName), SourceUser},
		{filepath.Join(workspace, ".dcode", ConfigFileName), SourceProject},
	}
	if requirements != "" {
		sources = append(sources, struct {
			path   string
			source Source
		}{requirements, SourceLocked})
	}
	for _, c := range sources {
		layer, ok, err := LoadFile(c.path, c.source)
		if err != nil {
			return nil, err
		}
		if !ok {
			// A missing user or project file is the normal case. A missing
			// requirements file that something explicitly named is not: "there
			// is no policy" and "the policy failed to load" are different
			// facts, and starting anyway would silently hand the user every
			// permission the administrator meant to withhold.
			if c.source == SourceLocked {
				return nil, fmt.Errorf(
					"config: the locked configuration at %s does not exist. "+
						"Point DCODE_REQUIREMENTS_FILE at the right file, or unset it — "+
						"starting without a policy that was asked for is not the same as having none",
					c.path)
			}
			continue
		}
		out = append(out, layer)
	}
	return out, nil
}
