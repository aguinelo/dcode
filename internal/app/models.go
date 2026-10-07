package app

import (
	"sort"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
)

// models answers which models a session in workspace can ask for: the one it
// gets when it asks for none, and every profile it can name.
//
// Resolved by the code that builds the session rather than beside it: the
// chain is optionsFor's, a name is applied by applyModelRequest, and the
// family, the transport and the window come from the provider buildProvider
// composes. A menu that resolved on its own would offer one thing and open
// another the day either side moved.
//
// An empty workspace answers for the configuration the daemon started with. A
// workspace that is not one, or whose configuration cannot be read, is refused
// saying why, as a session there would be: a menu read from the daemon's own
// configuration would offer models the project never chose.
func (d *Daemon) models(workspace string) (protocol.ModelsResponse, error) {
	opts := d.opts.Base
	if workspace != "" {
		ws, err := validWorkspace(workspace)
		if err != nil {
			return protocol.ModelsResponse{}, err
		}
		if opts, err = d.optionsFor(ws); err != nil {
			return protocol.ModelsResponse{}, err
		}
	}
	return modelsOf(opts), nil
}

// modelsOf is what opts offers: its own model, then each profile by name.
func modelsOf(opts Options) protocol.ModelsResponse {
	names := make([]string, 0, len(opts.Profiles))
	for name := range opts.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	out := protocol.ModelsResponse{
		Default:  choiceOf(opts.Model, opts),
		Profiles: make([]protocol.ModelChoice, 0, len(names)),
	}
	for _, name := range names {
		out.Profiles = append(out.Profiles, choiceOf(name, applyModelRequest(opts, name)))
	}
	return out
}

// choiceOf describes, under name, the session opts would build.
//
// A bundle no session could be built with — a model no family claims, a family
// or a transport this build does not have — is listed anyway, as not measured,
// with the refusal the session would give. Dropped, it would leave a person
// looking for the profile they wrote; called measured, it would claim
// measurements for a family nothing here can resolve, since Unmeasured only
// answers for families that exist.
func choiceOf(name string, opts Options) protocol.ModelChoice {
	// Composing a provider needs no key, and a description built without one
	// cannot hand it out.
	opts.APIKey, opts.CredentialFrom = "", ""
	c := protocol.ModelChoice{
		Name: name, Model: opts.Model,
		Family: resolvedFamily(opts), Transport: opts.Transport,
		BaseURL: opts.BaseURL, Window: opts.Window,
	}
	p, err := buildProvider(opts)
	if err != nil {
		c.Notice = err.Error()
		return c
	}
	c.Family, c.Transport = p.Family().Name(), p.Transport().Name()
	if c.Window == 0 {
		// The session's own arithmetic (New): the family's answer unless
		// configuration gave one, and no window said when the family has none.
		c.Window, _ = p.Window(opts.Model)
	}
	c.Notice = provider.Unmeasured(c.Family)
	c.Measured = c.Notice == ""
	return c
}
