# A recusa diz por quê

**Data:** 2026-10-01
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, §4 e invariantes)
**Fonte:** um `/loop` parado no primeiro segundo com "authentication rejected",
vindo do MiniMax

## O que mudou

Um 401 ou 403 de provedor chegava à pessoa como o texto seco "authentication
rejected", e um 402 como "quota or billing limit reached". O transporte lia até
8 KB do corpo e entregava a `ClassifyStatus`, que o jogava fora nesses dois
ramos — e só nesses: o 4xx genérico e o 5xx já levavam o corpo, sanitizado.

Os dois ramos mantêm a classe e agora acrescentam o motivo do provedor:
`authentication rejected: <motivo>`, `quota or billing limit reached: <motivo>`.
Corpo vazio deixa o texto fixo sozinho, sem dois-pontos pendurado.

## Por que a classe não bastava

A classe existe para o laço (RN-5), e para o laço chave inválida, revogada, de
outra região ou conta sem saldo são uma decisão só: parar. Para a pessoa são
consertos diferentes, e o texto do provedor é a única coisa que separa um do
outro. A uma chave que não existe, o endpoint do MiniMax compatível com OpenAI
responde 401 com:

    {"type":"error","error":{"type":"authorized_error","message":"login fail: Please carry the API secret key in the 'Authorization' field of the request header (1004)","http_code":"401"},"request_id":"…"}

Gravado de `api.minimax.io` em 2026-10-01, com uma chave inventada. O endpoint
compatível com Anthropic responde 401 no mesmo envelope, com
`authentication_error` e a mensagem apontando o cabeçalho `X-Api-Key`. Os dois
corpos estão no teste.

## Por que só o `error.message`

O envelope `{"error": {"message": …}}` é o dos dois dialetos que este pacote
fala, e o MiniMax o usa nos dois endpoints. A mensagem sai sozinha porque o resto
do envelope é envelope: a TUI mostra um erro numa linha só, cortada na largura do
terminal, e em 80 colunas o JSON em volta enchia a linha antes de o motivo
começar.

Fora desse envelope, o motivo é o corpo como veio, aparado. Adivinhar outros
campos seria inventar um formato que ninguém declarou, e o corpo inteiro não
perde nada.

## Credencial

O motivo passa por `sanitize`, como todo corpo que chega a uma mensagem: um
provedor recusando uma chave é justamente o que tende a citá-la de volta. A chave
registrada sai pelo valor — e é isso que cobre uma chave no formato do MiniMax,
sem `sk-` nem `Bearer` para o padrão reconhecer. As que nunca foram registradas
saem pela forma: `Bearer …`, `x-api-key: …`, `sk-…`. O lugar delas fica marcado
com `[redacted]`, para um motivo com um buraco não parecer inteiro.

A varredura com chave sentinela que sustenta a RN-6 olhava só o status 400.
Agora olha todo status cuja mensagem leva o corpo: 400, 401, 402, 403 e 500.

## Como fica preso

- `TestARejectionCarriesTheProvidersReason` — os dois corpos gravados do
  MiniMax, um 403, um 402, texto puro, envelope sem mensagem, corpo vazio e em
  branco. É o teste da invariante nova.
- `TestAKeyQuotedInARejectionIsRedacted` — a chave registrada no formato do
  MiniMax e chaves nunca registradas, dentro e fora do envelope, em 401, 402 e
  403.
- `TestHTTPTransportSaysWhyAKeyWasRejected` — o caminho inteiro pelo transporte
  HTTP, com o servidor citando de volta a chave que recebeu.

Os três foram vistos vermelhos antes da correção, com a mensagem seca no lugar do
motivo. E uma mutação de cada metade os põe vermelhos de novo: tirar o
`sanitize` reprova os testes de chave, e ler o corpo cru em vez do envelope
reprova os de motivo.

## O que fica de fora

O 4xx genérico e o 5xx continuam mostrando o corpo cru, envelope incluído, e 429
e 413 continuam sem corpo nenhum. Ler o motivo do mesmo jeito em todos os ramos é
outra mudança.
