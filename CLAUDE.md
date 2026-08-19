# dcode — Claude Code

The instructions for this repository live in `AGENTS.md`, which every agent that
works here reads, `dcode` included. This file is only for what is specific to
Claude Code.

@AGENTS.md

## Specific to Claude Code

- Named agents coordinate through `SendMessage`, not through polling or shared
  state. Spawn them in one message with `run_in_background: true`, give each a
  `name`, and tell each one who to message next.
- Give every writing agent its own worktree and a non-overlapping set of files.
  Read-only research may run concurrently and report back to whoever owns the
  files.
- Only the integration owner edits shared manifests and lockfiles, or reconciles
  overlapping changes.
- Swarm for work that spans three or more files, a new feature, a cross-module
  refactor, an API change, security or performance. Not for a single-file edit,
  a two-line fix, a documentation change or a question.

Ruflo, MCP and hook configuration is user-level, in `~/.claude/`. It is not
repeated here: a per-project copy is a copy that drifts, and this repository has
been bitten by exactly that.
