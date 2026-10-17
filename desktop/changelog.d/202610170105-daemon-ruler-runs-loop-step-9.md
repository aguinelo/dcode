## en

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

## pt-BR

- **A régua do daemon roda um loop (passo 9 da proposta do loop).** Dois
  cenários entram no `check:daemon`, que passa a ter 13: `loop-qualifies`
  começa um loop por "Nova sessão" → "↻ loop" numa pasta que não declara
  critério ("0 · qualifica antes"), vê a régua do loop na janela ler
  `qualifying`, o card do `done.proposed` esperar em `awaiting_review` sem nada
  trabalhado, "Começar o trabalho" abrir a sessão de trabalho, as luzes dos
  critérios andarem com o primeiro `done.checked` (✓ ola, ✗ tchau) antes de o
  turno terminar, e o veredito "1 pasta trabalhada · 3 de 3 critérios";
  `loop-stops` digita `/loop specs/ola` num campo (nenhum turno daquela sessão
  começa) e "Parar agora" interrompe o turno de trabalho e encerra o loop
  `stopped`, sem nada escrito. Cada passo é conferido contra o `/v1/loops` e o
  log de eventos do daemon, com o modelo roteirizado segurando as respostas
  entre as quais a janela é lida. Antes, nada conduzia um loop por um daemon de
  verdade a partir da janela. `tests/daemon/**` é caminho protegido do
  `docs/loop/done.toml`; o dono autorizou mudá-lo só neste PR (2026-10-15). O
  `docs/loop/tasks.md`, "O que a régua lê", também é protegido e não foi
  autorizado: os atributos que os cenários do loop leem (`data-pane-id`,
  `data-loop-id`, `data-loop-state`, `data-loop-review`) estão listados no topo
  de `tests/daemon/loop.mjs` até aquela tabela ser atualizada.
