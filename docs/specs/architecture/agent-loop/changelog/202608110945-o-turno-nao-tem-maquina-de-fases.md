# O turno não tem máquina de fases

**Data:** 2026-08-11
**Specs afetadas:** `202608072335-agent-loop` (`.p`, `.i`)

> **Regra:** o estado observável de um turno é o `SessionState` do protocolo —
> `idle`, `running`, `blocked`, `closed`. Não existe uma segunda máquina de
> estados por fase, e a spec deixa de descrever uma.

## O que existia

A seção 2 abria com um tipo `TurnPhase` e cinco constantes — `assembling`,
`streaming`, `executing`, `blocked`, `done` — e a seção de invariantes prometia:

> Nenhuma transição fora do diagrama da seção 2 é possível; transição inválida
> retorna erro, nunca `panic`.

O código declarava o tipo com outro nome (`Phase`, não `TurnPhase`) e **nada o
lia**. Nem o loop, nem o servidor, nem o cliente, nem um teste. Não havia função
de transição, então não havia transição inválida a recusar: o invariante
descrevia o comportamento de uma máquina que ninguém construiu.

Deriva em cima de código morto é o sinal de que ele estava morto desde cedo — o
nome do tipo divergiu da spec sem que ninguém notasse, porque ninguém tinha
motivo para escrever o nome.

## Por que sai em vez de ser implementado

É a mesma decisão tomada para as chaves de configuração declaradas e não lidas:
**o que não é lido hoje sai, e volta com código junto quando for preciso.**

Uma superfície declarada e inexistente é pior que ausente. Ela promete um
controle que não existe — aqui, a garantia de que o turno recusa uma transição
inválida — e quem lê a spec para entender o produto sai sabendo menos do que
entrou, com a confiança de saber mais.

E a fase não estava só morta, estava **duplicada**. `protocol.SessionState` é o
estado que o cliente realmente observa, aparece em vinte e cinco lugares do
código e é o que a TUI renderiza. Ter dois vocabulários para o estado de um
turno, um vivo e um morto, é a condição em que a próxima pessoa implementa o
morto por engano.

## O que não muda

`StopReason` fica inteiro. Ele é real, é retornado, e desde
`TestEveryStopReasonIsReachableFromTheLoop` existe um teste que falha se alguma
razão declarada deixar de ter quem a produza — que é exatamente a guarda que
faltava aqui.
