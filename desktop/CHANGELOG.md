# Changelog — desktop

🇧🇷 [Versão em português](CHANGELOG.pt-BR.md)

The desktop client's own record, versioned apart from the core: tags
`desktop-vX.Y.Z`, starting at 0.1.0. Nothing has been released yet. What changed
and why, one entry each; the detail of a decision is in `docs/DECISIONS.md`.

---

## Unreleased

- **A refused turn says the model refused.** A turn that ends with the core's
  new stop reason `refused` gets the line "Parou: o modelo recusou responder"
  after the error entry carrying the provider's words, instead of ending with
  no line of its own. The protocol types are regenerated for `StopRefused`.
- **PR and CI colours follow D52 (D61).** Running CI is neutral, not the
  waiting yellow, and a merged or closed PR is neutral, not green: D52 keeps
  yellow for what waits for you and green and red for what was measured. CI
  that passed stays green, failed red, an open PR orange.
- **The changes panel opens a PR and shows its CI, always confirmed (D60).**
  Crew's Mudanças tab and the grid's context card show the branch's PR — `#N`
  and title, opening in the browser, its CI as a dot, a word and counts, and
  whether it is open, a draft, merged or closed — or "Criar PR", held with the
  reason on a detached head or on the base itself. "Criar PR" opens a sheet
  prefilled with the conversation's title, its last answer and the files it
  wrote, and the repository's default base, all editable, beside the branch
  that will be pushed; it says what Confirmar does (`git push -u origin`,
  never forced, and the PR through gh; never a merge) and what stays out
  because it is not committed. Nothing is sent before Confirmar. Read when the
  panel opens, on ↻ and on focus at most every 30 s, and every 60 s while CI
  runs and the panel is shown. The owner's request and rule; sidebar badges
  are left out (a GitHub call per row, and they would change the measured
  screen).
- **`generated.ts` carries a branch's PR.** The core gains `GET /v1/git/pr`
  and `POST /v1/git/pr` (`GitPRResponse`, `GitPR`, `GitChecks`,
  `CreateGitPRRequest`) and the codes `gh_unavailable` and `pr_refused`, so the
  changes panel can show a branch's PR with its CI and open one a person
  confirmed. Regenerated; nothing in the app calls them yet.
- **PDFs and text files go with a turn, like pictures (D59, amending D46).**
  The composer's "+" offers pictures, PDFs and text files; pasting or dropping
  a file attaches it. The main process types each one by its bytes — a
  picture, then `%PDF-`, then UTF-8 without NUL, the extension only saying
  which text — and holds it to the daemon's rules from `generated.ts`: 10 MiB a
  PDF, 256 KiB of text, 5 documents a turn besides 20 pictures, a file name and
  never a path, checked again when the turn is sent. A document is a chip with
  its kind, name, size and ×; past a count it is refused by name. The message
  in the flow shows the documents `turn.started` names, so a conversation read
  back from its record shows them too. A PDF for a model that does not read
  one is refused by the daemon at submit, naming the model, and the window says
  so in the field and a notice, keeping the text and the files. Before, the
  "+" took pictures only. Screens 02 and 03 stay at 0.00%; `check:daemon` 13 of
  13.
- **`generated.ts` carries documents on a turn.** The core's
  `SubmitTurnRequest` gains `documents` (`TurnDocument`: name, media type,
  base64), `TurnStarted` gains `documents` (`AttachedDocument`: name, media
  type, size), and the accepted types and limits are constants. Regenerated,
  and the event decoder accepts `turn.started.documents` as optional; nothing
  in the app sends or draws them yet — the composer does in its own
  pull request.
- **The activity line shows what the turn touched (D58).** Next to the verb
  and the elapsed time, the files the running turn wrote (the writing tools'
  `path`, from the turn's start) show as small chips, and the pictures sent
  with its message as thumbnails; past five (three on a grid window) a "+N"
  names the rest and shows them. Clicking a file opens its diff under the line,
  the same detail its tool line opens. Before, what the turn wrote was read
  only by scrolling its tool lines, or in the Crew's panel, which sums the
  whole conversation. Screens 02 and 03 have nothing written on a running line
  and stay at 0.00%.
- **The plan stays pinned as a checklist while the turn runs (D57).** The
  plan the running turn last revised (`plan.updated`) stands above the flow,
  out of the scroll, and the flow leaves it out: a grid window gives it one
  line ("passo 2 de 3 · ..."), the maximized conversation and the Crew the
  whole checklist, with the step under way as a turning ring. When the turn
  ends it goes back into the flow where it was. Before, the plan scrolled out
  of view exactly while it moved. The plan entry now records the turn that
  revised it, since the event carries none. The recording has no plan;
  screens 02 and 03 stay at 0.00%.
- **The daemon ruler runs a loop (step 9 of the loop proposal).** Two
  scenarios join `check:daemon`, which now has 13: `loop-qualifies` starts a
  loop from "Nova sessão" → "↻ loop" on a folder that declares no criteria
  ("0 · qualifica antes"), sees the window's loop ruler read `qualifying`, the
  `done.proposed` card wait in `awaiting_review` with nothing worked, "Começar o
  trabalho" open the work session, the criteria lights move with the first
  `done.checked` (✓ ola, ✗ tchau) before the turn ends, and the verdict "1 pasta
  trabalhada · 3 de 3 critérios"; `loop-stops` types `/loop specs/ola` in a
  field (no turn of that session starts) and "Parar agora" interrupts the work
  turn and ends the run `stopped`, with nothing written. Each step is checked
  against the daemon's `/v1/loops` and event log, with the scripted model
  holding the replies the window is read between. Before, nothing drove a loop
  through a real daemon from the window. `tests/daemon/**` is a protected path
  of `docs/loop/done.toml`; the owner authorized changing it for this PR only
  (2026-10-15). `docs/loop/tasks.md`, "O que a régua lê", is also protected and
  was not authorized: the attributes the loop scenarios read (`data-pane-id`,
  `data-loop-id`, `data-loop-state`, `data-loop-review`) are listed at the top
  of `tests/daemon/loop.mjs` until that table is updated.
- **`generated.ts` carries the loop's new fields.** The core's `LoopRun` gains
  `args` and each `LoopSpecRun` what the survey found (`criteria`, `unmet`,
  `unavailable`, `error`) and what its proposal wrote (`proposal_criteria`,
  `proposal_summary`), so the loop sheet and the review card draw them without
  measuring again. Regenerated, and the loop decoder accepts them as optional
  (an absent `criteria` is zero); nothing in the app draws them yet.
- **The quit question counts live loops (D55, amending D19).** Quitting the
  app stops the daemon it started, and a loop lives in that daemon: it ends
  with it and does not come back. The question before quitting now counts the
  live loops next to the sessions at work and says so ("Fechar o DCode Box e
  encerrar 1 loop?"); before, a loop waiting for review with no session at
  work was ended without a word.
- **A proposed definition of done is reviewed before the work (D56).** While
  a loop waits in `awaiting_review`, its ruler shows a card drawn from
  `done.proposed` as the daemon measured it: the file, each criterion with its
  class (acceptance, regression, broken), what running it did, the command, the
  proposer's reason and the output; the proposer disagreeing with the run, a
  broken criterion and a set with nothing red are warned; protected paths are
  listed. A goal that qualified several folders gets one proposal each.
  "Abrir no editor" asks the main process, which opens only a file the daemon
  listed as that loop's proposal. "Começar o trabalho" continues the loop
  (`POST /v1/loops/{id}/continue`); when another client answered first,
  `loop_not_awaiting_review` is said with the daemon's reason. The Crew's
  Pronto tab gets a Loop section. Before, the review existed only in the TUI.
  Screens 02 and 03 have no loop and stay at 0.00%.
- **A loop is started and stopped from the window (D55).** `/loop
  <folder|goal> [--protect glob]` in any field goes to the daemon as typed
  (`POST /v1/loops`), and is in the "/" menu and the help; "↻ loop" in the new
  session window opens a "Começar loop" sheet with the folders the daemon lists
  and what each declares, or a goal, and what to protect, sending the same text
  a person would type. A loop takes one grid window that follows its current
  session and keeps its place from folder to folder (Q5 A, amending D31); a
  ruler at its top — only when there is a loop — says the goal, the state, the
  folder it is in of how many, each folder's phase one click from its
  conversation, the verdict, the error and the warnings, with "Parar agora" and
  "Parar depois desta pasta". The main process keeps the loops' stream next to
  the conversations'. Refusals (`loop_already_running` and the rest) are said
  with the daemon's reason, and a loop that fails is said once. Before, the
  daemon could run a loop and the window could neither start, see nor stop it.
  Screens 02 and 03 have no loop and stay at 0.00%.
- **Sessions and conversations are read with `loop_id`.** The core now runs
  `/loop` and marks the sessions a loop opens (step 4 of
  `docs/proposals/loop.md`); `generated.ts` is regenerated with the loop types,
  and the session and conversation decoders accept the field. Nothing in the
  app uses it yet: starting and following a loop are steps 7 and 8.
- **The criteria lights move with each check (D54).** While a turn runs, the
  lights in the grid window, the maximized one and the Crew's Pronto tab are
  those of its last `done.checked`, not of the last turn that ended, with
  "ciclo N" beside them and, once it stalls, "parado s de L" against the
  ceiling the event carries. When the turn ends, `turn.completed` takes over
  again. A check that broke a criterion and made the loop put work back says so
  in the flow ("⟲ Ciclo 2 voltou: e2e quebrou, desfeito src/cart.ts"); before,
  the lights stood still until the turn was over and a rollback was invisible.
  Screens 02 and 03 have no check under way and stay at 0.00%.
- **The desktop reads `done.checked`, and does not show it yet.** The core now
  announces every check of the definition of done inside a turn (step 3 of
  `docs/proposals/loop.md`), and a type the protocol adds fails this app's
  tests until it is decoded. It is decoded and validated against the core's
  golden files; the criteria lights still move at `turn.completed`, and moving
  them with each check is step 6, its own change.
- **`done.proposed` is read in the shape the core now emits.** The daemon
  announces a written proposal (step 2 of `docs/proposals/loop.md`) with the
  `done.toml` path and without `turn_id`, `round` and `expires_at`, since it
  waits for nothing. The decoder follows the regenerated `generated.ts`; the
  note in the stream is unchanged, and the review card comes in its own step.
- **The desktop no longer reads `done.signed`.** The core removed the event and
  `SignDoneRequest` (step 1 of `docs/proposals/loop.md`, decision Q1 C): the
  signature is the reviewed `done.toml`, and nothing ever emitted either. The
  regenerated `generated.ts` loses both, and the decoder and the
  "Definição de pronto assinada" note go with them; `done.proposed` stays.
- **A design for running `/loop` from the desktop, approved by the owner on
  2026-10-13, with every recommendation taken; it is built before the
  orchestrator.**
  `docs/proposals/loop.md` designs starting, following, reviewing and stopping
  a loop from the grid and the Crew, the core request it rests on (N4: the
  daemon runs the `/loop` sequence, so both clients see one loop, and an event
  per check inside a turn), and what to do with `done.proposed`,
  `done.signed` and `SignDoneRequest`, which nothing emits. Written before any
  code because the core half changes a versioned protocol and the owner asked
  to approve the design first; nothing in the app changes yet.
- **Run from the sources on macOS, the menu bar and the Dock say DCode Box.**
  `npm start` now first runs `scripts/dev-name.mjs`, which writes the name into
  the development Electron's Info.plist (in node_modules) and signs that bundle
  again ad hoc, since macOS reads those two places from the bundle and not from
  `app.setName`. It does nothing when the name is already there, and `npm ci`
  undoing it is fixed on the next start. A failure is said and never stops the
  app. A packaged app is untouched.
- **The app is called DCode Box and shows the mascot's icon when run from the
  sources.** Run with `npm start`, the binary is Electron's own, so the Dock and
  the taskbar showed Electron's icon. The icon is now set at runtime, and the
  product name (windows, dialogs, packaged bundle) is "DCode Box". On macOS the
  menu bar's app name and the Dock's label still read "Electron" in that mode:
  they come from Electron's own bundle (the entry above names it too).
- **dcode's colour is the mascot's Laranja.** The owner chose orange
  `#FF6A00` as dcode's main colour, replacing D29's lime `#c6f36b`; the
  neutrals and semantics come from the mascot sheet's 23 tones, in Grade and
  Crew, dark and light. Where a tone cannot be read as text, a step of the same
  hue stands in, with WCAG ratios in D52: the light accent is `#B33D00`
  (orange is 2.9:1 on white), the bright orange stays a fill under `#0A0A0A`
  ink (6.9:1), waiting moves to the sheet's yellow `#FDE68A` (dark) and
  `#7A5C00` (light) so it does not read as running, and red and green darken
  or lighten to stay above 4.5:1. The terminal's selection and ANSI hues and
  the window's first-frame ground follow. References 02 and 03 were
  recaptured by check:visual itself, as the owner authorized for this change,
  and both measure 0.00% (D52).
- **The mascot enters the app: the icon and each conversation's state.** The
  owner chose the mascot of `refs/design/brand/mascote.webp` as dcode's face.
  It is drawn by hand as vectors, never cropped from the sheet: the app icon
  (`assets/icon/icon.svg`, front pose in Laranja on a graphite tile) and
  `npm run gen:icons`, which rasterises it in Playwright's Chromium and packs
  the `.icns`, `.ico` and `.png` with no platform tool; `forge.config.ts` gives
  it to the packager. Where the window showed a conversation's state with a
  character — a grid window, the maximized one, Crew's thread and session
  rows — it now shows the mascot's head in the same tone, and the eye and a
  mark say the state again in shape: Planejando…, Codando…, Verificando…,
  Deu certo!, Bug encontrado…, Esperando você, Hum…, Parado. The pose is read
  from what the window already folds (the conversation's state, the phase of
  the call running, the last seal), never from a field the protocol lacks.
  Empty screens show the whole figure; notices carry the "Bug encontrado…"
  head. Reduced motion still stops the breathing. References 02 and 03 move
  by 0.07% and 0.04%, inside the limit; the palette change stacked on this
  one recaptures them once (D49).
- **A project's memory and skills are reachable from the grid.** Only Crew
  showed them. "Memória e skills…" in a conversation's ⋯ menu now opens, over
  the window, Crew's own Memória and Skills sections, a tab each, on that
  conversation's project (the project picker still switches). The same
  components, so what Crew says — what did not come, an older daemon — the
  grid says too. Offered mid-turn, since it only reads; Esc, a click outside
  or "Fechar" close it. Nothing at rest changes: both visual references
  measure 0.00%, which is why it lives in the ⋯ menu rather than as an
  always-visible entry (D53).
- **A session can be ended from the ⋯ menu.** The window only closed a
  session on its own (one never used, and the one left behind by a model
  switch), so a person could not end one. "Encerrar sessão" now closes it with
  the same `DELETE /v1/sessions/{id}`; the daemon keeps the record, the list
  shows the conversation ended, and writing in it continues it in a new
  session. In the middle of a turn, or with an approval waiting, the menu asks
  first, saying what would be cut short, with the focus on "Cancelar". A
  refusal is a notice with the daemon's reason. Nothing at rest changes: both
  visual references measure 0.00% (D51).
- **The commands the person wrote are offered and sent as in the TUI.** The
  TUI offers the slash commands in `commands/*.md`, the user's and the
  project's; the window could not read them. It now asks the core
  (`GET /v1/commands`) once per project it has a conversation of, as the TUI
  reads them once at start, and the `/` menu lists them after the field's own,
  with their description and whose they are. Typed, one is expanded as the
  TUI's `config.Expand` does (`$ARGUMENTS`, or what followed appended) and sent
  as a message; a file named like a field command never shadows it. The help
  lists them under "Deste projeto". A list that did not come is said, with the
  daemon's reason, at the foot of the menu, in the help and by an unknown
  `/name`; never an empty list. Nothing at rest changes: both visual references
  measure 0.00% (D50).
- **The core lists a workspace's user commands.** `GET /v1/commands` answers
  the slash commands the user and the project wrote, as the TUI finds them,
  each with the text it sends. What the field's `/` menu will offer beside its
  own commands, and send the way the TUI does. The protocol types are
  regenerated.
- **The field's commands are offered, and a help lists them with the keys.**
  The field answered `/undo`, `/compact`, `/mode` and `!`, and nothing said so.
  A `/` at the start of a conversation's field now opens a menu of every
  command, filtered as the name is typed (↑ ↓ choose, ⇥ completes, ↵ sends one
  that needs nothing more, esc closes), as the TUI's completion does. `/help`,
  or `?` in an empty field, opens a help over the window with the commands and
  the window's, the field's and an approval's keys. Two TUI commands join:
  `/init` sends the TUI's own DCODE.md turn, word for word (a test compares it
  with `internal/tui/commands.go`), and `/plan` shows the current plan, or with
  text asks for a new one as the TUI does. Both start a turn, so mid-turn the
  window refuses and says why instead of steering the work with another
  request. Nothing at rest changes: both visual references measure 0.00%
  (D48).
- **Images go with a turn.** The daemon has taken pictures on a turn since
  `SubmitTurnRequest.images`, and the TUI sends them; the window sent text only,
  and its "+" said attaching did not exist yet. Now "+" opens the system's
  picker (images only), a paste with an image on the clipboard attaches it, and
  files dropped on the field attach. The main process reads a picked file and
  checks pasted or dropped bytes the same way — the type from the bytes (PNG,
  JPEG, GIF, WebP), up to 10 MiB each and 20 per turn — and checks them again
  when the turn is sent; the renderer never sees a path. Thumbnails sit above
  the field, each with its ×, kept per conversation like the draft. A refusal
  is a notice and a line under the thumbnails, with the reason. A correction
  carries text only, so a message with images during a turn is held and said
  why, never steered or queued. The message in the flow shows the images it
  went with; the events carry none, so a conversation read from its record
  shows the text alone. Nothing at rest changes: both visual references
  measure 0.00% (D46).
- **The field remembers what was asked, and holds what comes next.** The TUI
  recalls past prompts with ↑/↓ and queues a message until the turn ends; the
  window did neither, so asking again meant retyping and every message during a
  turn became a steer. Now ↑/↓ in a conversation's field walk what it asked
  before, newest first, and give back the draft past the newest — only at the
  field's edges, so a message of several lines is still edited as text. The
  prompts come from the conversation's own `turn.started` and `turn.steered`
  events, so they survive a restart with no second store; a continued
  conversation reads its source's record the first time ↑ passes the oldest.
  During a turn ↵ still steers (D23) and ⌥↵ queues: the queue shows as chips
  above the field, each with ×, ⌃X drops the last one, and each is sent when
  the conversation is idle again, one per turn. A refused send stops the queue,
  says why and waits for "Retomar"; only messages are queued, a command is
  refused with the reason. Both visual references still measure 0.00% (D47).
- **A tool call opens onto what it did.** The flow showed each model call as
  one summary line, though `tool.completed` always carried the whole output
  and, for a call that changed a file, its unified diff; only a command typed
  with `!` was shown open. Now a click on the line, or focus and ↵/space,
  opens it: a change shows its diff with + and − coloured as in the git panel
  (the same renderer), anything else its output in monospace with ANSI codes
  stripped, and a failure its whole error. An output past 40 lines opens on
  the first 40 with "mostrar tudo"; a cut by the tool and the daemon's 400-line
  diff cut are said, never silent. Live sessions and recorded histories alike.
  Closed by default and drawn exactly as before: both visual references
  measure 0.00% (D45).
- **The window says where a turn stands against its ceilings, and what the
  last one cost.** The rounds and in-flight counts arrived and were folded, but
  nothing showed them, and of `turn.completed`'s `usage` only the context was
  kept. Now the composer bar — a grid window's foot and the maximized one's —
  shows "rodada N/M" from half the ceiling (warning, and error from three
  quarters) and "N/M em voo" when every slot is taken, as the TUI's bar does;
  far from the ceilings it draws nothing, and the whole count is on the stop
  button's hover. Crew's Sessão tab shows the running turn and, for the last
  one that ended, output, input summed across rounds, and cache. Tokens travel
  only at a turn's end, so nothing is estimated while it runs, and a turn
  without `usage` says the provider did not report it. Nothing at rest changes:
  both visual references measure 0.00% (D44).
- **A conversation can be renamed from the window.** The daemon has named
  conversations, live and ended, since `POST /v1/sessions/{id}/name`, and the
  window never asked: a title was whatever the first question was. Now a
  double-click on the title (a grid window, the maximized one, Crew's thread)
  turns it into a field, and "Renomear" in the ⋯ menu opens the same field.
  Enter or leaving the field saves, Esc gives up, an empty name gives the
  automatic title back, and the window's keys stay out of the field. The title
  changes when the daemon says so, not when the name is typed: a live
  conversation's `session.renamed`, or, for an ended one (whose rename no
  stream announces), its row read again from `GET /v1/conversations`. A refusal
  is a notice with the daemon's reason and the field keeps the text. The name
  goes through a new main-process channel: id in the core's alphabet, name
  trimmed and refused, not cut, past the core's 120 characters. Nothing at rest
  changes: both visual references measure 0.00% (D43).
- **A conversation that ended opens onto its history.** Opening one showed an
  empty thread, and Crew's right panel said a finished conversation carries no
  events, so the person took their sessions for lost. A recorded conversation
  without a live session, whether in a grid window, maximized or in Crew's
  thread, is now read page by page from the core's
  `GET /v1/conversations/{id}/events` through a new main-process channel (id
  checked against the core's alphabet, position from 1). Its events are folded
  like a live session's and drawn read-only by the same flow, above the same
  "Escrever continua a conversa numa sessão nova" field. Crew's Mudanças lists
  the files it changed. While the window reads it says so. A failure is shown
  in the thread with the daemon's reason and a retry, and as a notice; it never
  looks like an empty conversation. Past 20 pages the window says how much was
  left unread. Both visual references still measure 0.00% (D42).
- **Where a conversation works is clear.** The conversation in focus showed its
  title and the folder's short name, the branch from the session's start, and
  only what the tools said they wrote. Now, in Crew, under the title: the
  session's id, the folder, the branch now, whether the folder is a worktree and
  of which checkout, and the folder's +/−. Mudanças lists each file changed
  since the last commit with its lines, and opening one shows its diff; Sessão
  has the id and folder with copy, live or ended, the branch the session
  started on when the folder moved since, and the commit. In Grade, the branch
  chip and each window's foot open a card with the same, without changing what
  the grid draws (02 and 03 at 0.00%). Read from the daemon, by conversation,
  never by a path the window picks; a folder without git, git missing and an
  older daemon are said, never an empty list (D41).
- **The protocol types carry the page of a recorded conversation.**
  `src/protocol/generated.ts` is regenerated for the core's new
  `GET /v1/conversations/{id}/events` (`ConversationEventsResponse`,
  `ConversationEventsLimit`). Nothing in the window reads it yet; that is the
  next change, which opens a finished conversation onto its history instead of
  an empty thread.
- **The core reads a workspace's git as it is now.** Two routes answer, for a
  folder, the branch checked out now, the commit, whether it is a linked
  worktree and of which checkout, every file changed since the last commit with
  its lines, and one file's diff. What the context panel will read, instead of
  the branch frozen at session start and what the tool events said they wrote.
  The protocol types are regenerated.
- **An approval offers the answers the TUI offers.** The card always offered
  once, this session and deny; the protocol has five answers, and the TUI picks
  by the crossing. A network crossing now offers once, this project, always and
  deny (no "this session", as in the TUI); any other once, this session and
  deny. Keys 1 to n, ↵ and esc deny, deny stays preselected; the answers that
  are recorded beyond the session carry the warning tone. The daemon's reason
  is on the card, not only in a tooltip. Reference state 03 is a network
  crossing, so it changed with this (0.73%); the owner saw it and authorized
  recapturing 03 alone in the same PR, and both states measure 0.00% (D39).
- **Undo, compact and run a command from the window.** `/undo`, `/compact` and
  `!` were the TUI's alone, and text starting with `/` went to the model as a
  message. A conversation's ⋯ (a grid window, Crew's thread) now offers
  Desfazer o último turno — saying what went back and what stayed because it
  changed after the turn —, Resumir o contexto — saying when there was nothing
  worth compacting, which no event says —, and Rodar um comando…; `!comando` in
  the field runs it in the session and its output is shown open. `/undo`,
  `/compact` and `/mode <m>` work in the field; any other slash is "comando
  desconhecido" and stays there (D40).
- **A command the agent proposed can be run by the person, in a terminal below
  the conversation, and its result handed back.** Asked for after a denied
  approval the person wanted to run themselves, possibly with sudo. Every
  proposed command — the approval card's and each `bash` line in the flow —
  gets Copiar and Rodar no terminal, shown on hover or focus and out of the
  layout, so both visual references stay at 0.00%. Rodar no terminal docks a
  real shell (a PTY in the main process, `node-pty`, drawn by xterm.js) below
  the flow — in the grid window, the maximized conversation and Crew's thread —
  in the session's workspace as the daemon lists it, with the command typed and
  not run: the person edits it and presses Enter, and sudo asks for its
  password inside the terminal. zsh and bash mark their prompts (OSC 133), so
  when the command ends the window offers the result to the agent, showing the
  message first — command, exit code, the last 200 lines without ANSI, cut and
  said so, password-prompt lines dropped — and sends it only on a click, as a
  turn or a correction. Named channels checked in the main process; each
  terminal answers only the window that opened it; nothing of its stream is
  logged. Dependencies, exact: `node-pty` 1.2.0-beta.15 (1.1.0 ships macOS's
  spawn-helper without its exec bit), `@xterm/xterm` 6.0.0, `@xterm/addon-fit`
  0.11.0, loaded only when a terminal opens (D37).
- **A new session's model and mode are chosen before it opens.** Nova sessão
  always opened on the daemon's defaults, and changing either afterwards cost a
  new session or a turn in the wrong mode. The grid's "where?" window and Crew's
  "Nova ▾" gain "+ modelo e modo": on, choosing a project (or another folder)
  leads to the project's models as the daemon lists them and the four modes,
  and opens there. What is left on its default is not sent. The mode goes as
  the sandbox it maps to, and the window switches the mode right after when the
  session opened in another; a refusal leaves the session open and says why
  (D38).
- **A session's mode is switched from the window.** Plan, assist and auto were
  the TUI's alone (`/mode`, ⇧⇥); the window showed the mode and could not change
  it. The mode chip — in a grid window's field, the maximized conversation's and
  Crew's Sessão tab — opens a menu of the three, and ⇧⇥ cycles them in the
  conversation in focus. The chip changes when the session announces the switch,
  never before, so a refusal is never shown as accepted; it is said with the
  daemon's reason. Auto is drawn in the warning tone, as on the TUI's bar, and a
  pair that is none of the three is called custom rather than the nearest mode
  (D36).
- **A sober light theme, chosen per window; dark stays the default.** Grafite
  was dark only (D29), which leaves out daylight and anyone who reads better on
  light. Sistema (follows the system), Claro or Escuro, in Crew's
  Configurações and, on the grid, behind the ⚙ at the foot of the sidebar —
  which said "not yet" before and costs no pixel the rulers measure. Light is a
  token block for each look: a neutral near-white ground, near-black text, the
  faintest grey above 4.5:1 everywhere, an olive-lime accent for text and marks
  (the bright lime stays only as the send button's fill, under dark ink), and
  amber, green and red darkened to read as text. The colours still written in
  the styles became tokens. The window opens hidden until its first frame is
  painted, so neither palette flashes the other. Unset, it is Escuro, and both
  visual references still measure 0.00% (D35).
- **The model's text is drawn as the markdown it is.** Replies with bold
  headings and bullet lists showed literal asterisks and dashes. Model text now
  draws paragraphs, bold, italic, inline code, fenced code blocks (monospace,
  scrolling sideways), modest headings, bullet and numbered lists with nesting,
  quotes, rules, tables and links. `marked` 18.0.14 (MIT, no dependencies) only
  tokenises, and the tree is mapped to React elements: no HTML string is built,
  so HTML the model writes shows as text and the CSP is unchanged. Only `https`
  links are links, opened in the browser through the main process; any other
  address stays text with the address in its title, and images are their alt
  text. While a reply streams, an unclosed `**` stays as typed and an unclosed
  fence is code up to where the text has reached. Styles use only the existing
  tokens, so the grid, its compact windows and Crew all take them (D34).
- **The visual references are captures of the approved window.** The owner
  approved the new look — grafite, the grid's windows, Crew — and `check:visual`
  measured it against the handoff's mock, which it no longer resembles. The two
  measured states (02, 03) are now captured from the window by the check itself,
  in its own conditions; the floor is 0 px on both, and the limit stays 1.25%.
  A future visual change replaces them in its own PR after approval. Decisions
  D21 onward move to `docs/DECISIONS-2.md`, keeping each file under 500 lines
  (D18).
- **Crew's memory and skills types come from the core.** Written by hand while
  #432 was open, they are now the ones tygo generates from `internal/protocol`;
  `src/protocol/crew.ts` keeps only the decoder that checks an answer against
  them, so a change in the core's shape fails `check:protocol` instead of
  drifting (N6).
- **Crew becomes a dashboard with a rail of sections, on the daemon's data.**
  The first Crew put a third panel beside the grid's sidebar; studied against a
  dashboard of that kind, it lacked the shape that makes one: sections, and a
  list of sessions of its own. Crew now has a narrow icon rail — Painel, Agenda,
  Memória, Skills, Apps, Conhecimento, Configurações — and the Painel three
  panes: every conversation the daemon lists, searched and grouped (active, by
  project, older), the one in focus whole, and its context in tabs — Mudanças,
  Pronto, Sessão. Skills and Memória read the daemon's new routes (N6, #432):
  each skill with its source, file, triggers and what makes it ask first, and
  the files refused; each memory by kind, marking stale and unread ones, with
  the blocks that are not memories. A daemon older than the routes is said to
  not expose them, in its own words. Configurações shows the look, the daemon
  and each project's models. Routines, apps and knowledge have nothing in the
  core yet and say so instead of showing an empty list. The grid is untouched and
  stays the default. Icons from `lucide-react` (MIT) (D33).
- **A second version of the screen, Crew, beside the grid.** After Kiro Crew, and
  without taking the grid apart: a switch in the top bar alternates Grade and Crew,
  and the grid stays the default. Crew has three panels — the list, the
  conversation in focus whole, and on the right its seal and criteria, the files
  it changed, where it runs, and the rest of the crew — in high contrast (the
  faintest grey at about 6:1), with Geist and Fragment Mono; Kiro's Diatype is
  licensed and cannot ship. Fonts are now always their own files in the build, so
  the CSP never refuses one silently (D32).
- **Each window of the grid is a conversation you work in.** Asked for after
  seeing the grid: every quadrant an isolated window, like a conversation of its
  own. A window now holds its conversation whole — seal and criteria, the flow
  scrolling on its own, the approval and its own field — so several are worked on
  at once without opening any. One opened, clicked into or written in stays until
  its window is closed; what runs or waits keeps its place. Nova sessão asks where
  in a window of its own, offering the projects already known, and becomes the
  session there; with none known, the folder picker comes at once. Writing to an
  ended conversation continues it in the same window. ⌘1–9 puts the caret in
  window n, ⌘↵ maximizes and restores, Esc leaves a field (D31).
- **The window opens on an attention grid, in grafite.** One conversation at a
  time read as a chat app's clone, and a person running several agents wants to
  see them all. The window now opens on a grid of the conversations that claim a
  place — waiting for you, pinned, running, or finished and not yet seen — in
  places that do not move while the work does, up to 6 panels, or 9 in a large
  window. Each panel carries the seal of its last turn and each done criterion,
  the tail of its work, and an approval answered in place; ↵ opens it, with the
  other panels as peers in its head, and Esc or ⌘0 comes back. Grafite is dark,
  with one lime accent, the system's font and mono where it is code, and 4px
  corners; the seals and their words are in Portuguese. The handoff's visual
  references no longer describe the window, so `check:visual` fails against them
  until new ones are approved (D18, D29, D30).
- **A turn that wrote where the work is measured says so.** The daemon has long
  sent the protected paths a turn wrote (`touched_protected`), and the window
  never read them. The seal now says "· tocou a régua: …" and stops being green,
  and a panel says "⚠ tocou a régua" beside its criteria, with the paths in its
  title: a pass measured with a ruler the turn itself changed is not a plain
  pass (D13, D30).
- **A panel says when its model's family was never measured.** The seal says
  what the done criteria checked, not whether anyone measured the model doing
  the work. Beside the criteria, a panel and the open conversation say "sem
  medição" when the daemon's list of models says the family has no measurements
  behind it; a list that did not come marks nothing (D30).
- **The core lists a workspace's memory and skills.** Two routes answer, for a
  project, what its sessions read as memory — each entry, whether it is stale
  and whether a session shows it, and the blocks that are not memories, with
  why — and which skills they have, the user's and the project's, with where
  each comes from and whether it waits on a person. What the crew view will
  read, instead of parsing `.dcode/memory.md` and the skills folders with a
  copy of the core's rules. The protocol types are regenerated.
- **A conversation continues on another model.** The model button in the
  composer opens what the daemon says a session in that project can ask for
  (`GET /v1/models`): the project's default and each profile in `models.toml`,
  each marked measured — dcode's behavioural contracts were run against its
  family — or not, with the daemon's own warning. Picking one continues the
  conversation in a new session on it, with all of its history, and closes the
  one it left, whose record stays; left open, every switch would keep one more
  until the daemon refused new ones. While a turn runs or waits for an answer,
  the menu says to stop or wait. A daemon without the route says so in the menu,
  and no model is guessed in its place (D28).
- **A conversation that ends is no longer reported as a failure.** When a
  session closes, the daemon ends its stream and the list shows it ended; the
  window took the stream's end for a fault and said the events had stopped
  coming. It now waits a moment for the list, which arrives over a connection of
  its own, and says so only when the conversation is still live there.
- **The core lists the models a session can ask for.** One route answers, for a
  session's workspace, the model it gets by default and each profile it can
  switch to, each with its family and whether that family has measurements
  behind it — what the per-session model menu will read, instead of reading
  `models.toml` and keeping a copy of the core's list of measured families. The
  protocol types are regenerated.
- **A turn that goes round again speaks in a new message.** When the done check
  sends a turn back to work, the model's next words came out glued to its last
  ones — "…o parser.O teste…" — because streamed text joined any open message of
  the same turn. The daemon reports each new round with a progress event; text
  now joins only the message of the same turn and the same round.
- **⌘K finds any conversation.** The search the design draws, over the daemon's
  list: every word typed must be in the title or the project, without case or
  accents; the conversations running or waiting for you first, then one group per
  project; `tab` for the scope, the arrows for the cursor, `esc` to clear and then
  to close. `↵` opens the one under the cursor as a click on its row would, so an
  ended conversation is continued. The preview shows what the list carries — the
  state, the seal, the workspace, branch, model, turns and last event — and
  `docs/DECISIONS.md` records what the design asks for and the list does not
  (D27, L17).
- **The window runs on a real daemon.** Inside Electron, the main process
  attaches to the `dcode serve` answering on the socket — `DCODE_SOCKET`, else
  what `dcode socket` prints — or starts one there as its child, from
  `DCODE_BIN`, `~/.local/bin/dcode` or the `PATH`; it stops only the one it
  started, and asks first when quitting would stop a session that is running or
  waiting for you (D19). The bottom bar says connecting, connected with the
  daemon's version, lost or failed, with the reason. The sidebar is the daemon's
  list of conversations (N3), live and ended, by project: a live one opens its
  stream from the first event and resumes from the last when it drops; an ended
  one is continued in a new session the window opens (D21). The field sends a
  turn, corrects a running one (D23) and stops it; an approval is answered with
  `1`, `2`, `3`, `esc` and `↵`, which denies (D22); Nova sessão and ⌘N ask for a
  folder and open a session there. Every refusal shows the daemon's message, and
  nothing sent to a daemon that died looks accepted. The renderer still sees
  neither Node nor the socket: the preload hands it one named channel per request
  and per notice (`src/shared/api.ts`). A browser, or `?fixture=`, still shows
  the recording, and the visual check reads 1.20% and 1.21% — the numbers D22 and
  D23 were measured at.
- **The app keeps its data in a folder of its own.** Electron's default is the
  product name in the system's data folder, `~/Library/Application Support/DCode`
  on macOS — and the file system there does not tell `DCode` from `dcode`, the
  folder dcode keeps its configuration, records and state in. Every `npm start`
  wrote Chromium's storage, caches and cookies beside `models.toml`, and deleting
  the app's data would have taken dcode's configuration with it. The app now uses
  `dcode-desktop` there, and `DCODE_DESKTOP_USER_DATA` names another, absolute
  folder — which the check already passed and the app ignored. What earlier runs
  left in dcode's folder is Chromium's and can be deleted by hand; dcode's own
  entries there are `doctrine`, `models.toml`, `sessions` and
  `update-check.json`. `docs/DECISIONS.md` D26.
- **The loop now takes the window all the way to usable.** With N1–N3 in the
  core, `docs/loop/tasks.md` adds what day-to-day work needs on top of the
  connection: the sidebar listing every conversation, live and ended, from the
  core's list and its one stream; opening an ended one by continuing it in a new
  session; and ⌘K finding any conversation by its title. The ruler gains a
  scenario for each — `lists-recorded`, `continues-recorded`, `searches` — so
  `check:daemon` has eleven. Their daemon side was proven against a real
  `dcode serve` before they were committed; they fail on purpose, at the window,
  until it connects.
- **The core lists every conversation for the sidebar (N3).** One route with the
  live and the recorded conversations, each with title, project, state, branch,
  model, turns, seal, diff and last activity, and one stream of what changes —
  what the sidebar and the search will read instead of one stream per session.
  The protocol types are regenerated, and `docs/DECISIONS.md` marks N3 done.
- **A session the window opens reads its project's configuration (N2).** The
  daemon now resolves the configuration chain in each session's workspace, so a
  session opened in a project runs under that project's `.dcode/config.toml`,
  even with the daemon started outside every project, as the desktop starts it.
  A project configuration that cannot be read refuses the session with the
  reason. `docs/DECISIONS.md` marks N2 done.
- **The visual check stays out of CI for the reason that holds.** The documents
  kept it local because the reference images were not in the repository; they
  have been since the design handoff landed. Run once on the workflow's
  `ubuntu-latest`, it scores 1.76% and 1.79% against the 1.25% limit, identical
  on four runners: Linux's Chromium hints glyphs and antialiases them in LCD
  subpixels, where macOS — on which the limit was measured — and the references
  do neither. The limit stays: loosened to fit Linux, it would leave macOS more
  than 9,000 pixels of slack, room for a missing element to pass. `AGENTS.md`,
  the workflow, the ruler's note and the root changelog stop saying the
  references are missing, and `AGENTS.md` gives the reason that holds, with the
  launch flags that brought Linux to 1.19% and 1.20% — a ruler change, left for
  a decision of its own.
- **The core now says where the daemon is (N1).** `dcode socket` prints the
  daemon's socket path — one per user, whatever the environment — or fails with
  the reason, and the connection loop asks it instead of keeping a copy of the
  core's rule. `docs/DECISIONS.md` and `docs/loop/tasks.md` name the command.
- **The next loop connects the window to a real daemon, and its ruler is written
  before the code.** `npm run check:daemon` builds the app and `dcode` from the
  checkout and runs eight scenarios, each against a fresh `dcode serve` with a
  scripted model: attaching, starting its own, saying why it could not, a new
  session with a conversation, `↵` denying an approval, `1` allowing one,
  steering a running turn, and a daemon that dies. Each is checked against what
  the daemon logged, not only against what the window drew. `docs/loop/` now
  points `/loop` at it, with the ruler protected. It fails on purpose: the window
  does not connect yet, and each scenario says where it stops. Written first
  because a loop that writes its own ruler measures what it built, not what was
  decided.
- **Seven decisions for talking to the daemon, in `docs/DECISIONS.md` (D19–D25).**
  - The desktop starts the daemon when none answers, and that daemon stops when
    the app quits.
  - One daemon serves every project, with configuration resolved per session.
  - The session list comes from the protocol.
  - `↵` denies an approval and a message typed during a turn steers it, as in the
    TUI.
  - The loop is a screen over the mechanism that exists.
  - Live tokens wait.

  Taken before any wiring, because each one changes what the wiring is. What they
  need from the core is listed as N1–N4, each a core pull request with its spec
  first.
- **The desktop area exists, apart from the core.** Electron Forge with Vite,
  TypeScript and React, in `desktop/`, with its own README, changelog, version
  and `AGENTS.md`, and a stub `go.mod` so the core's `go … ./...` stops at the
  door instead of walking into `node_modules`. Separate because the desktop
  depends on the core and never the other way round, and a client that ships on
  its own clock should not borrow the core's version.
- **The main window of the v2 design, on recorded protocol events.** The sidebar
  of projects and sessions with their state marks, one session at a time with
  its flow — the question, compact tool lines, the model's text, the delegation
  card, the activity line — the approval card with its three answers and live
  clock, the composer and the bottom bar, in both themes, following the system.
  Recorded rather than live because the window had to be right before it is
  wired: the next version connects the main process to the daemon's socket. The
  bottom bar says `gravação` so a recording is never mistaken for a daemon, and
  every action that would need one says nothing was sent.
- **A pure reducer from protocol events to what the window shows**, tested with
  vitest. It reads only fields the protocol carries, validated at the boundary: a
  payload that does not match is said in the session's flow, a repeated event
  changes nothing, a missing one is named. Where the design asks for what the
  wire does not carry — a child's progress, live tokens, the approval's requested
  paths, the loop — it draws what exists and `docs/DECISIONS.md` lists the gap,
  because an invented field is a promise the daemon never made.
- **Protocol types generated from `internal/protocol`** with tygo, committed, and
  checked for staleness in CI; every event type the core declares must have a
  reader here, and every payload shape must agree with its generated type, or
  the tests and the typecheck fail. One definition of the wire, and a desktop
  that notices when it moves.
- **Project order, labels and collapsing are local preferences**, kept in the
  browser storage of the app and never sent to the daemon: collapse and expand,
  collapse all, rename inline, move up and down, and drag to reorder. Storage
  that cannot be read or written is said, and the sidebar falls back to its
  initial arrangement.
- **The renderer cannot reach anything.** Context isolation, sandbox, no Node, a
  preload that exposes the platform and the user's name and nothing else, a
  content security policy that lets it connect nowhere, fonts bundled, no port
  opened — and lint fails on a Node import in renderer code.
- **The design can be checked in a loop.** `npm run check:visual` compares the
  window in fixture mode with the design's reference screenshots (Playwright's
  Chromium, pixelmatch) and fails past a limit whose reasoning sits next to it:
  the design's own mock, rendered by the same Chromium, is 1.15% and 1.19% from
  the references, and the window measures 1.17% and 1.19%. `desktop/docs/loop/`
  lets dcode's own `/loop` iterate against it, with the references and the ruler
  protected. Local only: the limit was measured on macOS and holds there.
