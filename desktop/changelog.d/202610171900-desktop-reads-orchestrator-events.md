## en

- **The desktop reads the orchestrator's events at the boundary.** The ten events the core now
  declares for an orchestrator — `child.*`, `review.verdict`, `gate.measured`, `question.*`,
  `merge.*`, `main.health` — each have their payload's shape in `src/protocol/orchestrate.ts`, and
  a session's and a conversation's `parent_id`, `project_id`, `role` and `owns` are checked like
  every other field. A payload that does not match is said with the field, as before; one that
  matches changes nothing on screen yet, because the live agent list is a later step. Read now
  because an event the daemon can send must be one this desktop checks, never one it lets
  through unread; the test reads the core's own goldens, so a field the core renames fails here.

## pt-BR

- **O desktop lê os eventos da orquestradora na borda.** Os dez eventos que o núcleo passa a
  declarar para uma orquestradora — `child.*`, `review.verdict`, `gate.measured`, `question.*`,
  `merge.*`, `main.health` — têm o formato do payload em `src/protocol/orchestrate.ts`, e
  `parent_id`, `project_id`, `role` e `owns` de sessão e de conversa são conferidos como qualquer
  outro campo. Payload que não confere é dito com o campo, como antes; o que confere ainda não
  muda nada na tela, porque a lista viva dos agentes é um passo seguinte. Lido agora porque evento
  que o daemon pode mandar tem de ser evento que o desktop confere, nunca um que ele deixa passar
  sem ler; o teste lê os goldens do próprio núcleo, então campo que o núcleo renomeia falha aqui.
