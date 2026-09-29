# A contagem é do turno que roda

**Data:** 2026-09-29
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 7.2 e invariantes)
**Fonte:** conferência do handoff de design do desktop contra o código —
divergência 11 de `refs/design/desktop/CONFERIDO.md`, na branch
`docs/desktop-design-brief` (#400).

## O que mudou

`internal/tui/model.go`, tratamento de `EventTurnStarted`: o uso do turno —
`InputTokens`, `OutputTokens` e `CacheTokens` — é zerado junto com as rodadas e
os lugares em vôo. A linha de atividade continua desenhando a contagem que o
modelo tem; o que mudou é que o modelo só tem a do turno que roda. Até ele
relatar a sua, a linha diz o tempo e nada sobre tokens.

A seção 7.2 trocou a linha "Custo até aqui | `Usage` acumulado" por "Tokens | o
`Usage` que **este** turno relatou; até ele relatar, nada", e o exemplo perdeu o
`1.2k tok`. A seção 10 ganhou a invariante, reivindicada por
`TestANewTurnDoesNotShowTheLastTurnsTokens`.

## Por que o número não era do turno

O campo que a linha lia era escrito num lugar só: `turn.completed`, o único
evento do protocolo que carrega `Usage`. O `turn.started` zerava as rodadas, com
um comentário dizendo exatamente por quê — número de trabalho que não começou é
número em que se acredita — e deixava o uso de pé. O primeiro turno de uma
sessão rodava sem contagem; todo turno depois dele rodava sob a contagem em que
o anterior tinha terminado: `12.0s  4.3k tok` doze segundos dentro de um
trabalho que ainda não tinha produzido nada.

O teste que existia não via isso porque montava o modelo à mão, com
`OutputTokens` já preenchido e o turno já rodando. Conferia o que a linha
desenha a partir de um estado, nunca a sequência de eventos que produz esse
estado — e o defeito estava na sequência.

## Por que não uma contagem ao vivo

A spec prometia "custo até aqui", e isso o cliente não tem como saber: nenhum
evento entre `turn.started` e `turn.completed` diz quanto o turno gastou.
Estimar pelo texto que chega — caracteres divididos por quatro, ou um
tokenizador no cliente — daria um número que o daemon nunca mandou e que
discordaria do que o provedor cobra. É a mesma classe de defeito que o medidor
de contexto já teve, quando derivava do contador cumulativo em vez de ler o que
o daemon mediu.

Então, hoje, a contagem não aparece na linha: o relato do turno chega com o fim
dele, e a linha sai no mesmo evento. O código que a desenha fica, porque é o
mesmo que desenharia uma contagem do próprio turno se o protocolo algum dia a
mandar durante o turno. Esse evento é decisão de `client-server-protocol`, não
desta spec.

## Versão

Correção, não quebra de contrato. A seção 7.2 é `stable`, mas a linha que saiu
descrevia um campo que o protocolo nunca entregou, e o que a tela mostrava no
lugar dele era falso. Nada na superfície que a versão cobre se move — o
protocolo fica como estava —, então é `fix:`, PATCH.
