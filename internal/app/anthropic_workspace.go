package app

import (
	"strings"

	"github.com/aguinelo/dcode/internal/provider"
)

// workspaceHeader is what the Anthropic API reads the workspace from.
const workspaceHeader = "anthropic-workspace-id"

// explainWorkspaceRefusal turns the API's refusal of a key that is not scoped
// to a workspace into the sentence that fixes it.
//
// The API's own words name a header, and nobody can set a header in dcode, so
// a person reading them has nowhere to go. Any other refusal is returned as
// the provider said it. This reads a provider payload, not a Go error: the
// header name in the body is the only thing that says which refusal this is.
func explainWorkspaceRefusal(pe *provider.ProviderError, body, workspace string) *provider.ProviderError {
	if pe == nil || pe.Class != provider.ErrClassBadRequest || !strings.Contains(body, workspaceHeader) {
		return pe
	}
	if workspace != "" {
		pe.Message = "the Anthropic API refused the workspace " + workspace +
			" set in model.anthropic_workspace (DCODE_ANTHROPIC_WORKSPACE): " + pe.Message
		return pe
	}
	pe.Message = "this Anthropic key is not scoped to a workspace, so every request has to name one. " +
		"Set model.anthropic_workspace in config.toml, or DCODE_ANTHROPIC_WORKSPACE, " +
		"to the workspace id (wrkspc_...) from the Anthropic console. The API said: " + pe.Message
	return pe
}
