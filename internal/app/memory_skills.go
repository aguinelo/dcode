package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aguinelo/dcode/internal/behavior"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/memory"
	"github.com/aguinelo/dcode/internal/protocol"
)

// skillMaxBytes is the largest skill file a session loads.
const skillMaxBytes = 256 << 10

// skillDir is one place skills are found, and whose it is.
type skillDir struct{ source, dir string }

// skillDirs are where a session in workspace finds its skills, in load order:
// the user's, then the project's, which wins on a name.
//
// One list, read by New and by the route that lists skills, so the list cannot
// offer a skill the session does not find.
func skillDirs(roots config.Roots, workspace string) []skillDir {
	return []skillDir{
		{protocol.SkillSourceUser, filepath.Join(roots.Config, behavior.SkillsDirName)},
		{protocol.SkillSourceProject, filepath.Join(workspace, ".dcode", behavior.SkillsDirName)},
	}
}

func dirsOf(ds []skillDir) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.dir
	}
	return out
}

// whose names the source of a file found under one of ds, and its path
// relative to that directory. The last match wins, as the last directory wins
// on a name.
func whose(ds []skillDir, path string) (source, rel string) {
	source, rel = "", filepath.Base(path)
	for _, d := range ds {
		r, err := filepath.Rel(d.dir, path)
		if err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			source, rel = d.source, r
		}
	}
	return source, rel
}

// workspaceOptions resolves the configuration a session in workspace would
// get, refusing a workspace that is not one, or whose configuration cannot be
// read, with the reason a session there would be refused with.
func (d *Daemon) workspaceOptions(workspace string) (Options, error) {
	ws, err := validWorkspace(workspace)
	if err != nil {
		return Options{}, err
	}
	return d.optionsFor(ws)
}

// memory answers what a session in workspace reads as memory.
func (d *Daemon) memory(workspace string) (protocol.MemoryResponse, error) {
	opts, err := d.workspaceOptions(workspace)
	if err != nil {
		return protocol.MemoryResponse{}, err
	}
	return memoryOf(opts), nil
}

// memoryOf reads the memory of opts.Workspace by the calls New makes: Read for
// the file, knownCommits for staleness, and the cap Render applies. Every entry
// is listed, and shown says which ones the prefix carries.
func memoryOf(opts Options) protocol.MemoryResponse {
	max := opts.MemoryMax
	if max <= 0 {
		max = memory.DefaultMax
	}
	out := protocol.MemoryResponse{
		Path: memory.FileName, Enabled: opts.Memory, MaxEntries: max,
		Entries: []protocol.MemoryEntry{}, Malformed: []protocol.MemoryMalformed{},
	}
	if _, err := os.Stat(memory.Path(opts.Workspace)); !os.IsNotExist(err) {
		out.Exists = true
	}
	f, err := memory.Read(opts.Workspace)
	if err != nil {
		// A session there opens without its memory and says why; so does this.
		out.Unreadable = err.Error()
	}
	known := knownCommits(opts.Workspace, f)
	hidden := memory.Hidden(f, max)
	for i, e := range f.Entries {
		out.Entries = append(out.Entries, protocol.MemoryEntry{
			Kind: string(e.Kind), Subject: e.Subject, Body: e.Body,
			Learned: e.Learned, Commit: e.Commit,
			Stale: memory.Stale(e, known),
			Shown: opts.Memory && err == nil && i >= hidden,
		})
	}
	for _, line := range f.Malformed {
		out.Malformed = append(out.Malformed, protocol.MemoryMalformed{Line: line, Reason: memory.Diagnose(line)})
	}
	return out
}

// skills answers which skills a session in workspace has available.
func (d *Daemon) skills(workspace string) (protocol.SkillsResponse, error) {
	opts, err := d.workspaceOptions(workspace)
	if err != nil {
		return protocol.SkillsResponse{}, err
	}
	return skillsOf(opts)
}

// skillsOf loads the skills of opts.Workspace from the directories New loads
// them from, with the same cap. A held skill is listed as held: whether it
// loads is a person's answer to a question the session asks, and a list has
// nobody to ask.
//
// A directory that cannot be read fails, as it fails the session.
func skillsOf(opts Options) (protocol.SkillsResponse, error) {
	env := opts.Env
	if env == nil {
		env = os.Getenv
	}
	roots, err := config.DiscoverRoots(env)
	if err != nil {
		return protocol.SkillsResponse{}, err
	}
	dirs := skillDirs(roots, opts.Workspace)
	loaded, err := behavior.LoadSkills(dirsOf(dirs), skillMaxBytes)
	if err != nil {
		return protocol.SkillsResponse{}, err
	}
	out := protocol.SkillsResponse{
		Enabled: opts.Skills, Skills: []protocol.SkillInfo{}, Notices: []protocol.SkillNotice{},
	}
	info := func(s behavior.Skill, held bool) protocol.SkillInfo {
		source, rel := whose(dirs, s.Path)
		return protocol.SkillInfo{
			Name: s.Name, WhenToUse: s.WhenToUse, Triggers: s.Triggers,
			Source: source, Path: rel, Held: held, Claims: s.Claims,
		}
	}
	for _, s := range loaded.Loaded {
		out.Skills = append(out.Skills, info(s, false))
	}
	for _, s := range loaded.Held {
		out.Skills = append(out.Skills, info(s, true))
	}
	sort.SliceStable(out.Skills, func(i, j int) bool { return out.Skills[i].Name < out.Skills[j].Name })
	for _, n := range loaded.Notices {
		source, rel := whose(dirs, n.Path)
		// The reason names the file by its absolute path; the notice already
		// says whose it is and where under that, which is all a reader needs.
		out.Notices = append(out.Notices, protocol.SkillNotice{
			Source: source, Path: rel, Reason: strings.ReplaceAll(n.Reason, n.Path, rel),
		})
	}
	return out, nil
}
