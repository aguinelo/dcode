# Compactar por pedido

**Data:** 2026-09-15
**Specs afetadas:** `202608072333-context-engine` (`.config`, seção 1), `202608072335-agent-loop` (`.p`)
**Fonte:** pedido do usuário — "vamos pensar, na reciclagem de contexto, um
auto compact quando o contexto chegar a 80%... isso deve ser habilitado ou
não pelo usuário, vem por default habilitado, mas o usuário pode desabilitar
e usar o `/compact` manualmente, inclusive ele não está implementado".

## O que já existia, e não mudou

O auto-compact em 80%, avisando o usuário, já existia por completo —
`ce.Plan`, `CompactAt` default `0.80`, o aviso que aparece na tela como
"Compacted"/"N mensagens resumidas". Nada disso foi tocado.

## O que mudou

Duas coisas, as que faltavam:

**1. `DCODE_COMPACTION_ENABLED` / `compaction.enabled`**, booleano, default
`true`. Desligado, o gatilho automático não dispara mais sozinho —
`Engine.maybeCompact` é o único lugar que lê este campo, e só ele.

**2. `/compact`**, comando de sessão em andamento. Força compactação **na
hora**, ligado ou não o interruptor acima. Rota nova,
`POST /sessions/{id}/compact` (`202608072240-client-server-protocol.p.spec.md`
§4), síncrona — nada de fila pro próximo turno.

## Por que forçar não lê o mesmo interruptor

`Engine.Compact` (exportado; `forceCompact`, a recuperação quando o provedor
recusa por contexto grande, hoje só chama `Compact` por baixo) **nunca**
consulta `cfg.Disabled`. O interruptor desliga o gatilho de 80% — a
compactação que dispara sozinha. Uma pessoa digitando `/compact`, ou um
provedor recusando a requisição por estourar o contexto real, não são o
gatilho automático: são pedido direto, um da pessoa e outro do fio. Desligar
"compactar sozinho" nunca foi pedido pra também significar "recusar quando
alguém pede".

## Por que dá pra fazer sem mutex novo

`Engine.session` (o histórico) nunca teve mutex próprio — de propósito,
porque nada além do laço do turno deveria tocá-lo enquanto um roda. `SetMode`
já resolveu esse problema uma vez, com `pendingPrompt`: o que não pode ser
aplicado no meio de um turno espera o próximo. Compactação forçada segue o
caminho que `Exec` (`!comando`) já abriu, mais estrito ainda —
`Session.Compact`: mesmo lock que `Submit`/`Exec` usam, mesma recusa quando o
estado não é `idle` (`CodeTurnAlreadyActive`), síncrona. Nenhum turno em
andamento, nenhuma corrida — e é exatamente o caso comum: ninguém digita
`/compact` no meio de uma resposta que está chegando.

## O que a resposta HTTP carrega, e por quê

`CompactResult{Compacted bool}`. `EventSessionCompacted` já alcança um
cliente conectado do jeito de sempre — mas uma sessão sem nada que valha a
pena resumir não produz evento nenhum, e a pessoa que acabou de pedir merece
saber disso em vez de ficar sem resposta. É a mesma pergunta que motivou a
correção de `202609151400-o-turno-que-parava-mudo.md`, num canto diferente:
o que a ausência de evento não consegue dizer, algo tem de dizer.

## O que ficou de fora

Nenhuma UI nova pro caso comum — quando algo é compactado, `/compact` fica
quieto do lado do cliente, porque o evento que já existe já conta a história.
A nota só aparece quando não havia o que compactar, ou quando a chamada foi
recusada (turno em andamento, sessão fechada) — os dois casos que o canal de
evento nunca cobriu.
