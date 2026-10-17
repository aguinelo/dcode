## en

- **`dcode decisions` reads the orchestrator's decision log, without the daemon.** It opens
  `<orchestrator.dir>/<project-id>/decisions.jsonl` directly — the log has to be readable when what
  failed was the daemon — with the project resolved from `--project` (default: here) the way the
  orchestrator resolves it: every path into the same tree, a linked worktree included, is the same
  project. `--child`, `--kind` and `--since` filter; `--follow` keeps printing as lines are written;
  `--json` returns the lines byte for byte. A line that does not read, a `seq` with a gap, or a torn
  last line is said on stderr with its line number, the rest is still printed, and the command
  exits 1; a missing log is an error naming where it looked. The command never writes to the log.
  This is step 11 of the orchestrator's first slice.

## pt-BR

- **`dcode decisions` lê o log de decisões da orquestradora, sem o daemon.** Abre
  `<orchestrator.dir>/<project-id>/decisions.jsonl` direto — o log tem de ser legível quando o que
  falhou foi o daemon —, com o projeto resolvido a partir de `--project` (padrão: aqui) como a
  orquestradora o resolve: todo caminho para a mesma árvore, worktree ligada inclusive, é o mesmo
  projeto. `--child`, `--kind` e `--since` filtram; `--follow` continua imprimindo conforme as
  linhas são escritas; `--json` devolve as linhas byte a byte. Linha ilegível, `seq` com buraco ou
  última linha rasgada é dita em stderr com o número da linha, o resto é impresso mesmo assim, e a
  saída é 1; log que não existe é erro dizendo onde foi procurado. O comando nunca escreve no log.
  É o passo 11 da primeira fatia do orquestrador.
