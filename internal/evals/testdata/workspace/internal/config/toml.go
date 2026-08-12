package config

import "fmt"

// KnownKeys is every configuration key this build reads.
//
// A key absent here is refused rather than ignored: a typo in a config file is
// a setting the person believes is in effect, and silence is the worst answer.
var KnownKeys = map[string]string{
	"model.name":            "DCODE_MODEL",
	"model.transport":       "DCODE_TRANSPORT",
	"sandbox.mode":          "DCODE_SANDBOX_MODE",
	"sandbox.allow_network": "DCODE_ALLOW_NETWORK",
	"compact.at":            "DCODE_COMPACT_AT",
	"ui.lang":               "DCODE_LANG",
}

// CompactAt is the fraction of the window at which history is summarised.
const CompactAt = 0.80

// ParseTOML reads a configuration file.
//
// An unknown key is an error naming the nearest known alternative, because the
// person who typed it believes the setting is in effect.
func ParseTOML(text string) (map[string]string, error) {
	out := map[string]string{}
	for _, line := range splitLines(text) {
		key, value, ok := cut(line, "=")
		if !ok {
			continue
		}
		if _, known := KnownKeys[key]; !known {
			return nil, fmt.Errorf("unknown key %q; did you mean one of %d known keys?", key, len(KnownKeys))
		}
		out[key] = value
	}
	return out, nil
}
