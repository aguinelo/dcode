package app

import (
	"os"
	"path/filepath"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
)

// commands answers which user-defined slash commands a client in workspace
// offers: the ones the TUI resolves there, by the same config.DiscoverCommands
// and the same cap, so a command cannot mean one thing in one client and
// another in the other.
//
// A file that cannot be a command fails the list, as it fails the TUI's start:
// a list without it would offer the rest as if the broken one did not exist.
func (d *Daemon) commands(workspace string) (protocol.CommandsResponse, error) {
	opts, err := d.workspaceOptions(workspace)
	if err != nil {
		return protocol.CommandsResponse{}, err
	}
	env := opts.Env
	if env == nil {
		env = os.Getenv
	}
	roots, err := config.DiscoverRoots(env)
	if err != nil {
		return protocol.CommandsResponse{}, err
	}
	set, err := config.DiscoverCommands(roots, opts.Workspace, config.CommandMaxBytes)
	if err != nil {
		return protocol.CommandsResponse{}, err
	}
	out := protocol.CommandsResponse{Commands: []protocol.CommandInfo{}, Overrides: []string{}}
	for _, name := range set.Names() {
		c := set.Commands[name]
		source := protocol.CommandSourceUser
		if c.Source == config.SourceProject {
			source = protocol.CommandSourceProject
		}
		// The file by its name under its commands directory: the directories
		// are flat, and the absolute path would carry where the user's
		// configuration lives.
		out.Commands = append(out.Commands, protocol.CommandInfo{
			Name: c.Name, Description: c.Description, Source: source,
			Path: filepath.Base(c.Path), Body: c.Body,
		})
	}
	out.Overrides = append(out.Overrides, set.Overridden...)
	return out, nil
}
