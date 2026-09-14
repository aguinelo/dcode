package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ModelsFile is where named model profiles live.
//
// A separate file from config.toml, for the same reason requirements.toml is
// separate: the shape does not fit the bijective key-to-variable schema
// ParseTOML checks. A profile's name is data the user chose, not a key this
// product declared — exactly the case ParseSections exists for.
const ModelsFile = "models.toml"

// profilePrefix is what a section header must start with. Anything else in
// this file is a mistake worth naming rather than a profile with a strange
// name — the file has one shape, and only one.
const profilePrefix = "profile."

// Profile is one named, ready-to-use model configuration: everything
// app.Options needs to build a Provider, bundled under a name a person chose
// so `/model <name>` can switch to all of it at once instead of only the
// model string.
type Profile struct {
	Name string
	// Model is the only required field. A profile naming nothing to run is
	// not a profile.
	Model string
	// Family, Transport and BaseURL are optional, same as their DCODE_*
	// counterparts: empty lets the model's prefix resolve a family the
	// ordinary way.
	Family    string
	Transport string
	BaseURL   string
	// Window overrides what the family reports, same meaning as
	// DCODE_WINDOW. Zero means unset.
	Window int
}

// LoadModels reads models.toml from root. An absent file is an empty map, the
// ordinary case for anyone who has not defined a profile.
//
// A file that exists but cannot be parsed is an error, never silently empty:
// a typo that is swallowed is the configuration bug this product refuses
// everywhere else, and a profile file is no exception.
func LoadModels(root string) (map[string]Profile, error) {
	data, err := os.ReadFile(filepath.Join(root, ModelsFile))
	if os.IsNotExist(err) {
		return map[string]Profile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: cannot read %s: %w", ModelsFile, err)
	}

	doc, err := ParseSections(string(data), ModelsFile)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	out := map[string]Profile{}
	for _, section := range doc.Order {
		if !strings.HasPrefix(section, profilePrefix) {
			return nil, fmt.Errorf("config: %s: section %q must be named %q, e.g. [profile.qwen-local]",
				ModelsFile, section, profilePrefix+"<name>")
		}
		name := strings.TrimPrefix(section, profilePrefix)
		if name == "" {
			return nil, fmt.Errorf("config: %s: [%s] names no profile", ModelsFile, section)
		}
		fields := doc.Values[section]
		model := strings.TrimSpace(fields["model"])
		if model == "" {
			return nil, fmt.Errorf("config: %s: profile %q has no model", ModelsFile, name)
		}
		p := Profile{
			Name:      name,
			Model:     model,
			Family:    fields["family"],
			Transport: fields["transport"],
			BaseURL:   fields["base_url"],
		}
		if w := strings.TrimSpace(fields["window"]); w != "" {
			n, werr := strconv.Atoi(w)
			if werr != nil || n < 0 {
				return nil, fmt.Errorf("config: %s: profile %q has a window of %q, want a non-negative integer",
					ModelsFile, name, w)
			}
			p.Window = n
		}
		out[name] = p
	}
	return out, nil
}

// MergeModels layers project profiles over user profiles, by name.
//
// Same direction as every other layer in this product: the project is more
// specific, and a profile it declares under a name the user also used is the
// project's own answer, not a collision to report.
func MergeModels(user, project map[string]Profile) map[string]Profile {
	out := make(map[string]Profile, len(user)+len(project))
	for k, v := range user {
		out[k] = v
	}
	for k, v := range project {
		out[k] = v
	}
	return out
}
