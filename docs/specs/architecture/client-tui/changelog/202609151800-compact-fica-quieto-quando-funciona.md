# `/compact` fica quieto quando funciona

**Data:** 2026-09-15
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 8)
**Fonte:** pedido do usuário, na mesma conversa que corrigiu o turno mudo —
`/compact` manual, "imediato", não enfileirado pro próximo turno.

## O que mudou

`/compact`, comando novo. Chama `Transport.Compact` (rota
`POST /sessions/{id}/compact`), síncrono. O detalhe é o que aparece na
tela em cada caso:

- **Compactou algo:** nada aparece por causa deste comando — `session.compacted`
  já chega pelo canal de sempre, a mesma nota "Compacted" que o gatilho
  automático já produz. Uma segunda nota diria a mesma coisa duas vezes.
- **Não havia o que compactar:** uma nota diz isso — o único caso que o
  evento nunca cobre, porque não dispara nada quando não há nada a fazer.
- **Recusado** (turno em andamento, sessão fechada): uma nota carrega o
  motivo, nunca `errMsg`. `errMsg` encerra o cliente inteiro — é o canal do
  stream em si falhando, não de um comando recusado. Ouvir "há um turno
  rodando" é coisa ordinária de acontecer; não é razão pra sair da sessão.

## Por que a distinção `errMsg` vs. nota importa

`errMsg` já existia com um propósito específico: `p.fatal = msg.err.Error();
return p, tea.Quit`. Usá-lo pro `/compact` recusado teria fechado o cliente
toda vez que alguém tentasse compactar com um turno rodando — o oposto de
"imediato", que foi o que se pediu.

## O que não mudou

O mecanismo de compactação em si — `Engine.Compact`, `Session.Compact`,
`ce.Plan` — vive inteiro em
`docs/specs/architecture/context-engine/changelog/202609151700-compactar-por-pedido.md`.
Esta entrada é só sobre o que a tela faz com o resultado.
