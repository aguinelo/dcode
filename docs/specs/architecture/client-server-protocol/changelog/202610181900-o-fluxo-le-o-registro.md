# O fluxo lê o registro como a reposição

**Data:** 2026-10-18
**Specs afetadas:** `202608072240-client-server-protocol` (`.p` §7 e §9).

## O que mudou

`GET /sessions/{id}/events?from=N` com `N` abaixo do que a sessão guarda em
memória passa a ler essa parte do registro da sessão, quando há registro, e
seguir ao vivo dali — o que `EventLog.Replay` já fazia. Sem registro, a
resposta continua `410 events_expired`.

## Por quê

Depois de reiniciar o app e o daemon, com um orquestrador e seus filhos
reabertos sob o próprio id (#519), o desktop mostrou um aviso por sessão:
"os eventos pararam de chegar … events before 1 are no longer held; the
earliest available is 600". A sessão reaberta começa a memória depois do
registro (`firstSeq = LastSeq + 1`), e o desktop a acompanha de `from=1`.

O #519 prometia que "um cliente anexando de 1 recebe a sessão inteira" e
testou a promessa em `Replay`. Nenhuma rota usa `Replay` para o fluxo: a rota
de eventos usa `Subscribe`, que recusava abaixo do horizonte sem olhar o
registro. Os dois passam a montar o atraso pelo mesmo código, de modo que
reposição e fluxo não divergem de novo.

Corrigir no daemon, e não no cliente, porque a config já diz que, com
gravação ligada, a retenção não é horizonte duro, e porque o defeito vale para
todo cliente: o desktop, o TUI e `pkg/client`. Fazer o cliente ler o começo
pela rota da conversa gravada e emendar com o fluxo seria repetir em cada
cliente a emenda sem buraco e sem repetição que `Subscribe` já faz sob uma
trava só — e a rota da conversa numera a cadeia inteira de 1, o que não é,
em geral, a numeração da sessão viva.

`experimental`, abaixo de 1.0. Uma resposta que era recusa passa a ser o
fluxo: **MINOR**.
