## en

- **A session can be ended from the ⋯ menu.** The window only closed a
  session on its own (one never used, and the one left behind by a model
  switch), so a person could not end one. "Encerrar sessão" now closes it with
  the same `DELETE /v1/sessions/{id}`; the daemon keeps the record, the list
  shows the conversation ended, and writing in it continues it in a new
  session. In the middle of a turn, or with an approval waiting, the menu asks
  first, saying what would be cut short, with the focus on "Cancelar". A
  refusal is a notice with the daemon's reason. Nothing at rest changes: both
  visual references measure 0.00% (D51).

## pt-BR

- **Uma sessão pode ser encerrada pelo menu ⋯.** A janela só fechava uma sessão
  por conta própria (a nunca usada, e a deixada para trás ao trocar de modelo),
  então a pessoa não tinha como encerrar uma. "Encerrar sessão" agora a fecha
  com o mesmo `DELETE /v1/sessions/{id}`; o daemon guarda o registro, a lista
  mostra a conversa terminada, e escrever nela a continua numa sessão nova. No
  meio de um turno, ou com uma aprovação esperando, o menu pergunta antes,
  dizendo o que se perderia, com o foco em "Cancelar". Uma recusa vira aviso com
  o motivo do daemon. Nada em repouso muda: as duas referências visuais medem
  0,00% (D51).
