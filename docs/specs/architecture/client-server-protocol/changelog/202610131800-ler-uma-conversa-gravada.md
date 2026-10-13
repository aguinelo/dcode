# Ler uma conversa gravada, sem carregá-la

**Data:** 2026-10-13
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: uma rota em §4,
um tipo em §5.2, cinco linhas novas em §9)
**Fonte:** relato da pessoa no cliente desktop: abrir uma conversa terminada
mostrava a thread vazia, e ela achou que as sessões tinham se perdido.

## O problema

O protocolo lista as conversas gravadas (`GET /v1/conversations`), mas a única
rota de eventos é `GET /sessions/{id}/events`, que serve sessão carregada. Uma
conversa que terminou tem registro em disco e nenhum fluxo: o cliente que a
abria não tinha como mostrar o que aconteceu nela, e mostrava nada. Continuar a
conversa só para vê-la abriria uma sessão que ninguém pediu.

## O que mudou

- **`GET /v1/conversations/{id}/events`** devolve uma página da conversa
  gravada, lida do registro, sem carregar nem continuar nada.
  `ConversationEventsResponse`: `events`, `total` e `next`.
- **A leitura é a do `Carry`**: a cadeia que o registro continua primeiro, da
  mais antiga para a mais nova; sem `session.created` e sem pergunta de
  aprovação já decidida. É o que uma sessão que continuasse a conversa
  reproduziria, então a janela mostra o mesmo antes e depois de continuar.
- **Uma sequência só, sob o id da conversa.** Cada registro da cadeia conta a
  partir de 1; lidos como estão, dois eventos teriam o mesmo número e um
  cliente que guarda o que já tem descartaria o outro. O `seq` da resposta é a
  posição nesta leitura, e é ela que `from` e `next` contam.
- **Paginada e com teto.** `limit` vai de 1 a `ConversationEventsLimit`
  (5000), que é também o default. Acima do teto, `from` zero ou valor que não é
  número é **recusado** com `invalid_input` e o que foi enviado — nunca
  aparado: um cliente que recebe menos do que pediu sem ser avisado toma a
  página pelo todo. `next` diferente de zero é a resposta dizendo que cortou.
- **O id fica dentro da pasta de registros.** Um id vira nome de arquivo, então
  `..` ou uma barra fariam dele um caminho. `session.ValidID` aceita só letras,
  dígitos, `-` e `_`, até 128 caracteres — um alfabeto fechado, em vez de uma
  lista do que é perigoso. A mesma regra vale para o id que o marcador
  `session.resumed` nomeia, que é dado lido do disco: marcador que sai da pasta
  não é seguido, como um ancestral que falta.
- **Erros ditos.** Registro que não existe é `session_not_found`, nunca uma
  conversa vazia. Daemon sem pasta de registros recusa com `invalid_input`
  dizendo que não guarda transcrições, como já faz ao nomear.

## Por que uma rota nova, e não `/sessions/{id}/events` para as duas

A rota de sessão é um fluxo SSE com `from` em `Seq` do log vivo, `410` para o
que a retenção cortou e ping a cada 20s. Uma conversa gravada não tem log, nem
retenção, nem nada para chegar depois: responder pelo mesmo caminho mudaria o
que um `404` ali quer dizer — hoje, "a sessão acabou" — para clientes que já o
leem assim. A conversa gravada é da família `/conversations`, que já é a que a
lista mostra.

## As invariantes

As cinco linhas novas em §9 são reivindicadas por
`TestARecordedConversationIsReadWholeAndInOneSequence`,
`TestAnIdThatLeavesTheRecordDirectoryIsRefused`,
`TestAMarkerThatLeavesTheRecordDirectoryIsNotFollowed`
(`internal/session`), `TestARecordedConversationIsReadPageByPage` e
`TestADaemonWithoutRecordsRefusesToReadOne` (`internal/server`).

Rota e tipo novos, nenhum contrato existente mudado: MINOR. O marcador que sai
da pasta deixar de ser seguido muda o `-c`, mas só para um registro que
ninguém escreveu pelo daemon.

## O que fica de fora

- **A TUI ler conversa gravada pela rota**, em vez de ler o disco: mudança da
  TUI, no PR dela.
- **Ancestral que falta** continua lido como conversa mais curta, sem aviso
  (decisão do `Carry`, de 202608250200). Dizer na resposta que a cadeia tem
  um buraco é outra mudança.
- **Nomear** (`POST /sessions/{id}/name`) ainda junta o id ao caminho sem
  passar por `ValidID`. A correção é a mesma linha, e é outro PR.
