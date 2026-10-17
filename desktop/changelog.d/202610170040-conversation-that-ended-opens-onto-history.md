## en

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

## pt-BR

- **Uma conversa terminada abre no histórico dela.** Abrir uma mostrava a
  thread vazia, e o painel da direita do Crew dizia que uma conversa terminada
  não traz os eventos dela: a pessoa achou que as sessões tinham se perdido.
  Uma conversa gravada sem sessão viva, numa janela da grade, maximizada ou na
  thread do Crew, agora é lida página por página da rota
  `GET /v1/conversations/{id}/events` do núcleo, por um canal novo do processo
  principal (o id conferido no alfabeto do núcleo, a posição a partir de 1).
  Os eventos são dobrados como os de uma sessão viva e desenhados só para
  leitura pelo mesmo fluxo, acima do mesmo campo "Escrever continua a conversa
  numa sessão nova". As Mudanças do Crew listam os arquivos que ela mudou.
  Enquanto lê, a janela diz que está lendo. Se a leitura falha, o motivo do
  daemon aparece na thread com "Tentar de novo" e também num aviso; nunca
  parece uma conversa vazia. Passando de 20 páginas, a janela diz quanto ficou
  sem ler. As duas referências visuais continuam a 0,00% (D42).
