## en

- **A reopened orchestrator is live in the list, and an orchestrator's sessions are never
  continued.** After a daemon restart the orchestrator came back under its own id, but the list of
  conversations still read it as ended: a reopened session writes no second `session.created`, and
  that was the only event that started a live row. The desktop then continued it, as D33 says for
  an ended conversation, in a new ordinary session with no role and no orchestration tools — which
  went on to make up actions. `Conversations.Reopened` now puts the reopened session back among the
  live rows, folded from its record, before its first new event. And `CreateSession{resume}` refuses
  the record of an orchestrator, a child or a reviewer with `invalid_input`, saying where the work
  goes on, whichever client asks; `dcode -c` skips them when it picks the newest conversation.

## pt-BR

- **A orquestradora reaberta é viva na lista, e as sessões da orquestradora nunca são
  continuadas.** Depois de um reinício do daemon a orquestradora voltava sob o próprio id, mas a
  lista de conversas ainda a lia como terminada: uma sessão reaberta não grava um segundo
  `session.created`, e só ele começava uma linha viva. O desktop então a continuava, como a D33 diz
  de uma conversa terminada, numa sessão comum nova, sem papel e sem as ferramentas de orquestrar —
  que passou a inventar ações. `Conversations.Reopened` agora põe a sessão reaberta de volta entre
  as vivas, dobrada do registro, antes do primeiro evento novo. E `CreateSession{resume}` recusa o
  registro de uma orquestradora, de um filho ou de um revisor com `invalid_input`, dizendo por onde
  o trabalho segue, seja qual for o cliente; o `dcode -c` os pula ao escolher a conversa mais nova.
