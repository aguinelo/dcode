# Uma sessão pode ter mãe

**Data:** 2026-10-17
**Specs afetadas:** `202608072240-client-server-protocol` (`.p` §5, §5.1, §5.2 e §8);
`202610132000-orchestration` (`.p` §5.2–§5.4; `.i`, passo 2).

## O que mudou

`Session` e `Conversation` ganham `parent_id`, `project_id`, `role` e `owns`, todos `omitempty`.
`role` é `orchestrator`, `child` ou `reviewer`, e `ValidRole` recusa qualquer outro valor — vazio
inclusive, porque a sessão sem papel se lê pela ausência do campo, nunca por um valor. A lista de
conversas lê os quatro do `session.created`, pelo mesmo `fold` que já lia `branch` e `loop_id`;
um filho gravado continua dizendo de quem é depois que o daemon que o rodou se foi.

`internal/protocol/orchestrate.go` passa a ter os tipos da orquestração (`Project`, `Child`,
`Return`, `Verdict`, `Finding`, `GateResult`, `ChildPR`, `MainHealth`, `Question`,
`QuestionOption`), os pedidos e respostas das rotas da `.p` §5.1, os dez eventos da §5.3 com os
payloads, e `errors.go` os treze códigos da §5.4, cada um com o status em `httpStatus`. O
`pkg/client` ganha as sete chamadas das rotas. **Nenhuma rota responde ainda**: o daemon as
atende a partir do passo 5; até lá, cada chamada recebe a recusa de rota inexistente, dita como
qualquer outra.

Dois nomes que a `.p` da orquestração não dava: o payload de `review.verdict` é `ReviewVerdict`
e o de `gate.measured` é `GateMeasured`, cada um com `child_id` e o `Verdict` ou o `GateResult`
achatado ao lado — `verdict.findings`, não `verdict.verdict.findings`. `MergeCompleted.PR` é o
número do PR, zero no merge local.

`TurnStarted.Origin`, `OriginChildren` e `StopDaemonRestart`, que a `.i` lista no passo 2, entram
com o passo 3 (reabrir uma sessão gravada sob o próprio id), que não compila sem eles e é feito em
paralelo.

## Por quê

O que faz uma sessão ser filha de outra tem de estar no registro dela, e não só na memória do
daemon: a orquestradora e os filhos voltam do reinício pelo id (passo 3), e a lista precisa saber
de quem cada um é sem perguntar ao daemon que caiu. Pôr os tipos no fio antes de qualquer rota
responder deixa os passos seguintes — log, projeto, filho, esteira — trocarem só comportamento,
e o desktop ler os eventos na borda desde já.

`experimental`, abaixo de 1.0. Aditivo: o golden de uma `Session` sem os campos novos é o de
antes, byte a byte. **MINOR**.

`Conversation` deixa de ser comparável com `==` em Go, porque `owns` é uma lista; quem comparava
linhas (`sameRow`, e dois testes) passa a comparar `owns` como lista e o resto com
`reflect.DeepEqual`.
