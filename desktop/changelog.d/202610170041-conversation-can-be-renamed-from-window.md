## en

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

## pt-BR

- **Uma conversa pode ser renomeada pela janela.** O daemon nomeia conversas,
  vivas e terminadas, desde `POST /v1/sessions/{id}/name`, e a janela nunca
  pedia: o título era o que a primeira pergunta fosse. Agora um duplo clique no
  título (janela da grade, a maximizada, a thread do Crew) vira um campo, e
  "Renomear" no menu ⋯ abre o mesmo campo. Enter ou sair do campo salva, Esc
  desiste, um nome vazio devolve o título automático, e as teclas da janela
  ficam fora do campo. O título muda quando o daemon diz, não quando o nome é
  digitado: o `session.renamed` de uma conversa viva ou, para uma terminada
  (cujo nome nenhum stream anuncia), a linha dela relida de
  `GET /v1/conversations`. Uma recusa vira um aviso com o motivo do daemon, e o
  campo guarda o texto. O nome passa por um canal novo do processo principal:
  id no alfabeto do núcleo, nome aparado e recusado, não cortado, acima dos 120
  caracteres do núcleo. Nada em repouso muda: as duas referências visuais medem
  0,00% (D43).
