# Roadmap

What is not built and why it would be worth building. Five items, each with the
reason, the shape, what already exists to build on, and what would break.

Nothing here is a commitment to a date. The order at the end is a
recommendation, not a schedule.

The first four came from asking what a mature agent harness has that this one
does not, and keeping only the answers that fit what dcode is trying to be. The
fifth came from using it.

---

## 1. Undo a turn

**The largest gap, and the one that fits this project's own reasoning best.**

`edit` refuses to touch a file it has not read, fails on an ambiguous match,
checks the hash, and shows a diff. All of that protects against the *wrong*
edit. Nothing protects against the *right* edit under a wrong decision — the
agent changes seven files competently and the third should never have been
touched.

Today the only way back is git, and only if you committed first.

This belongs here rather than in a prompt because it is a **Layer 1**
guarantee: it does not depend on the model behaving, exactly like the
verification seal and like a background process dying with its session.

### Shape

A snapshot of every file the turn is about to touch, taken at the moment the
turn starts, kept for the session, restorable as a unit.

- Content addressed by hash, so ten turns touching the same unchanged file cost
  one copy.
- Scoped to the workspace and to paths the policy already permits: a snapshot
  must not become a way to copy files the agent could not otherwise read.
- Restoring is itself an act with consequences, so it declares what it will
  overwrite and refuses when the file changed on disk after the snapshot —
  the same invariant `edit` already enforces, for the same reason.

### Already there

`tools.State` records every path read and written, with a write generation
counter (`WriteSeq`) the done-check already uses to tell "the suite passed"
from "the suite passed on this code". That is the hook: the set of files a turn
touched is known without asking.

The session record (`internal/session/record.go`) says what happened. This says
how to go back. They are the two halves of the same idea.

### What would break

Nothing in the loop. The risk is in the client: an undo the user cannot see
the extent of is worse than no undo, so the diff of what restoring would change
has to be shown *before* it happens, not after.

### Verified by

A turn that edits three files, an undo, and byte-identical files. Plus the
refusal path: a file changed on disk after the snapshot is not silently
overwritten.

---

## 2. The model can see a screenshot

`ce.Message` carries `Text` and nothing else. There is no path for an image to
reach the model, in either direction.

This is not decoration for a terminal tool. In the session that produced this
roadmap, three screenshots were the fastest way to explain a problem: a
rendering artefact in the TUI, a cut-off glyph, and a stuck process. A coding
agent that owns a terminal interface will be shown pictures of that interface
misbehaving.

### Shape

`ce.Message` gains content parts — text and image — rather than a second field,
because a message is a sequence and "text plus attachments" loses the order.
Each provider family maps parts to its own wire format, which is exactly what
`Formulation` already exists to isolate.

- A cap on bytes per image and per turn, declared when it truncates, like every
  other output in this codebase.
- Images from the client only. A tool that reads an image off disk is a
  separate decision with a separate blast radius.

### Already there

The family abstraction. Adding a content kind touches each adapter once and
nothing above them.

### What would break

`ce.Assemble` must stay pure and its output must stay byte-identical for the
same input, so an image has to be addressed by content, never by a path or a
handle that varies between runs. Token estimation also has to account for
images or the budget bands silently drift.

### Verified by

The purity test that already exists, extended to a message with an image. And
one contract: given a screenshot of an error, the model refers to what is in it
rather than asking for the text.

---

## 3. Reaching the network, through the permission that already exists

`tool-suite.r` puts web search and HTTP requests out of scope, and the reason
was sound when it was written: a network tool without a permission model is a
hole.

**That premise is gone.** The permission model was built for something else and
is now the strongest part of the product: per-project consent persisted in
`grants.toml`, a sandbox that blocks at the operating system, and a policy that
treats network as an axis orthogonal to paths.

Reading a library's documentation or a dependency's changelog is ordinary
programming work, and dcode cannot do it.

### Shape

One tool that fetches a URL and returns text, not a browser.

- Every call declares `Network: true`, which is the request the policy already
  understands.
- Content type restricted to text and markup; a binary body is refused rather
  than decoded into the context window.
- The response carries its source URL and its truncation, because a model that
  cannot tell a quote from a summary will produce both.

Search is a second decision. Fetching a URL the user or the code named is
narrow and useful; searching is a different capability with a different failure
mode, and it can wait for evidence that fetching is not enough.

### Already there

`policy.Request.Network`, the standing grant store, and the sandbox that
enforces it. This is wiring a tool to machinery that is already load-bearing.

### What would break

The eval harness refuses shell commands so scenarios cannot execute; a network
tool needs the same treatment, or contracts start depending on whatever
happened to be online that afternoon.

### Verified by

A denied grant means no request leaves the process — asserted at the boundary,
not by reading the tool's code. And the doctrine's rule that a refusal is final
applies here without amendment.

---

## 4. Continue the last session

"Nothing survives what created it" is a decision on record, and it stands: no
fleet of agents, no session that outlives the client.

Continuing is a different thing. It is reopening, here and now, what you were
doing — the same workspace, the same thread, deliberately, on purpose.

It became possible on the day the session record landed. The events are on
disk; the model history can be rebuilt from them.

### Shape

`dcode --continue` reopens the most recent session for this workspace.
`dcode --resume <id>` names one.

- The history is reconstructed from the record, not stored twice. A second
  copy would drift from the first, and this codebase has found that drift four
  times already.
- What cannot be restored is stated rather than faked: background processes
  died with their session, and approvals are consent given in a moment that has
  passed. Both are re-asked, not assumed.
- Reconstruction runs through the same compaction the live session uses, so a
  continued session is not shaped differently from one that never stopped.

### Already there

The record, the append-only log, and `contextengine` assembling history from
messages. What is missing is the map from events back to messages.

### What would break

The record becomes load-bearing rather than merely useful, which raises what a
failure to write costs. Today an unrecorded session is a session nobody can
audit; then it is also a session nobody can continue, and the difference has to
be visible before someone relies on it.

### Verified by

A session, a continue, and a model that answers a question that only makes
sense given the earlier turns.

---

## 5. The input box holds more than one line

Reported from use: there is no way to break a line while typing. A list, or one
answer per line, cannot be written — everything collapses into one paragraph,
because Enter sends.

**Shift+Enter should insert a newline. Enter still sends.**

### The terminal problem, which is the whole difficulty

Most terminals send the same bytes for `Enter` and `Shift+Enter`. Distinguishing
them needs an enhanced keyboard protocol the terminal has to support and the
program has to request.

So this cannot be one binding. It needs:

- The enhanced protocol requested at startup, and used where the terminal
  reports it.
- A fallback binding that works everywhere — `alt+enter`, and `ctrl+j`, which
  is a literal newline on every terminal ever made.
- `/help` showing whichever one is actually live, because a shortcut documented
  and not working is worse than one nobody mentioned.

### The layout problem, which is where the damage would be

`BodyHeight` reserves rows by arithmetic:

```go
h := g.Height - 3          // status, input, bottom bar
```

The `3` assumes the input is exactly one line. A box that grows without that
number growing with it paints over the transcript — which is precisely the
ghosting already fixed once, in "a painted frame owns every cell it covers".

So:

- `renderInput` and `BodyHeight` must read the **same** function for the box's
  height. Two places computing it is the bug, not the symptom.
- The box grows to a cap — around ten rows — and scrolls inside itself past
  that. A pasted essay must not eat the window.
- `InputCursor` is a rune offset into a flat string today. Up, down, home and
  end become line-aware, and home on a wrapped line means the start of the
  line, not the start of the buffer.
- Bracketed paste of multi-line text inserts newlines instead of sending once
  per line.
- The queue prefix `(2) >` and the right-aligned scroll hint still have to
  land, on the first and last row respectively.

### Verified by

A golden render at several box heights, asserting the transcript ends exactly
where the box begins — the assertion that would have caught the original
ghosting. Plus the fallback bindings under a terminal that reports no enhanced
keys.

---

## Not doing, and why

**MCP.** A large surface with its own lifecycle, auth and failure modes.
Staying out was right and still is.

**Project hooks.** Commands run on tool events. Powerful, and exactly how
configuration becomes undergrowth: the session that produced this roadmap ran
under another product's hooks injecting noise into every turn.

**Sub-agents that write.** Three of the four known problems with delegation
exist *because* the child writes — conflict between children, inherited
approval, and undoing. Read-only makes all three disappear, and that reasoning
has not weakened.

---

## Suggested order

| | why here |
|---|---|
| **5** — multi-line input | Smallest, and it is daily friction reported from real use. |
| **1** — undo | Largest reduction in real risk, and structural rather than prompted. |
| **2** — images | Real workflow gap, evidenced; contained to the message type and the adapters. |
| **4** — continue | Newly possible, and cheap once the record is trusted. |
| **3** — network | Most valuable per line of code, and the one that most changes what dcode is. Last because it deserves the most deliberate decision. |
