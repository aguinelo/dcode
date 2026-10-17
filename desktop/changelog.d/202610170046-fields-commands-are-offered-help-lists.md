## en

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

## pt-BR

- **Os comandos do campo aparecem, e uma ajuda os lista com as teclas.** O
  campo atendia `/undo`, `/compact`, `/mode` e `!`, e nada dizia isso. Um `/` no
  começo do campo de uma conversa agora abre um menu com todos os comandos,
  filtrado enquanto o nome é digitado (↑ ↓ escolhem, ⇥ completa, ↵ envia o que
  não pede mais nada, esc fecha), como a completação da TUI. `/help`, ou `?` no
  campo vazio, abre por cima da janela uma ajuda com os comandos e as teclas da
  janela, do campo e da aprovação. Dois comandos da TUI chegam: `/init` manda o
  turno do DCODE.md da própria TUI, palavra por palavra (um teste compara com
  `internal/tui/commands.go`), e `/plan` mostra o plano atual ou, com texto,
  pede um novo como a TUI pede. Os dois começam um turno, então no meio de um a
  janela recusa e diz por quê, em vez de redirecionar o trabalho com outro
  pedido. Nada em repouso muda: as duas referências visuais medem 0,00% (D48).
