## en

- **A recorded conversation can be read without loading it.**
  `GET /v1/conversations/{id}/events` answers one page of a conversation that
  ended, read from its record — the chain it continues first, numbered as one
  sequence under its own id, the way a session continuing it would replay it.
  Pages are capped at 5000 events; a larger `limit` is refused rather than
  trimmed, and a page that stops short says where to read on. The id must be
  letters, digits, `-` and `_`, so it cannot name a file outside the record
  directory, and a `session.resumed` marker naming one outside it is no longer
  followed. A missing record is `session_not_found`, never an empty
  conversation. Before, the only route for events served loaded sessions, so
  the desktop opened a finished conversation onto an empty thread and the
  person took it for lost.

## pt-BR

- **Uma conversa gravada pode ser lida sem ser carregada.**
  `GET /v1/conversations/{id}/events` responde uma página de uma conversa que
  terminou, lida do registro — a cadeia que ela continua primeiro, numerada
  como uma sequência só sob o id dela, como uma sessão que a continuasse a
  reproduziria. A página tem teto de 5000 eventos; `limit` maior é recusado em
  vez de aparado, e página que para antes do fim diz onde continuar. O id só
  pode ter letras, dígitos, `-` e `_`, então não nomeia arquivo fora da pasta
  de registros, e um marcador `session.resumed` que nomeia um fora dela deixa
  de ser seguido. Registro inexistente é `session_not_found`, nunca conversa
  vazia. Antes, a única rota de eventos servia sessão carregada, e o desktop
  abria uma conversa terminada numa thread vazia, que a pessoa tomou por
  perdida.
