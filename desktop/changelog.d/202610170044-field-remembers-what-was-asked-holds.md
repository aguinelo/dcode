## en

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

## pt-BR

- **O campo lembra o que se pediu e guarda o que vem depois.** A TUI traz de
  volta o que foi pedido com ↑/↓ e põe uma mensagem na fila até o turno acabar;
  a janela não fazia nenhum dos dois, então pedir de novo era redigitar, e toda
  mensagem durante um turno virava redirecionamento. Agora ↑/↓ no campo de uma
  conversa percorrem o que ela já pediu, da mais nova para a mais velha, e
  devolvem o rascunho depois da mais nova — só nas bordas do campo, então uma
  mensagem de várias linhas continua editada como texto. O que se pediu vem dos
  próprios eventos `turn.started` e `turn.steered` da conversa, e sobrevive a
  reiniciar o app sem um segundo armazenamento; uma conversa continuada lê o
  registro da de origem na primeira vez que ↑ passa da mais velha. Durante o
  turno ↵ continua redirecionando (D23) e ⌥↵ põe na fila: a fila aparece em
  fichas acima do campo, cada uma com ×, ⌃X tira a última, e cada uma é enviada
  quando a conversa volta a ficar parada, uma por turno. Um envio recusado para
  a fila, diz por quê e espera "Retomar"; só mensagens entram, e um comando é
  recusado com o motivo. As duas referências visuais seguem medindo 0,00% (D47).
