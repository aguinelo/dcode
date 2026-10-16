# O prefixo Anthropic é lido do cache

**Data:** 2026-10-15
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, seções 3 e 7);
`202608072335-agent-loop` (`.p`, seção 8)
**Fonte:** a família `claude` pagava preço cheio pelo contexto inteiro a cada
rodada, e Crew → Sessão no desktop mostrava o cache sempre em zero.

## O que mudou

No dialeto anthropic o cache de prompt só existe se a requisição pedir, com
`cache_control` num bloco. Nenhuma requisição pedia. Agora a família `claude`
marca três pontos `ephemeral`, abaixo do limite de quatro da API:

- a última ferramenta, que guarda todas as definições — fixas desde o início
  da sessão;
- o system, que guarda ferramentas e system juntos;
- o último bloco da última mensagem, que guarda a conversa até ali.

Como o contexto é append-only (ADR-03), a rodada seguinte reenvia esse prefixo
byte a byte e o lê do cache. A marca não entra nos bytes guardados, então
mudá-la para o novo último bloco a cada rodada não invalida prefixo nenhum —
não é edição de mensagem já enviada.

O system passa a ir como lista de blocos quando carrega a marca. Sem marca
continua string, que é o que `minimax-m3` manda no mesmo dialeto: nada diz que
o endpoint da MiniMax lê a marca, e mandar o que não foi medido é o que este
pacote evita.

E o uso passa a ser lido onde está. O `message_start` traz a entrada e o que o
cache leu e escreveu; o `message_delta` traz a saída. O decoder lia só o
segundo, e por isso `cache_read_input_tokens` e `cache_creation_input_tokens`
chegavam sempre em zero. Agora guarda o primeiro e combina no `EventDone`.

O laço somava os tokens lidos do cache e jogava fora os escritos, no turno e
no filho delegado. Agora soma os dois — a escrita é a metade que custa mais
que um prompt sem cache, e a que a tela não mostrava.

## Por que três e não quatro

A quarta marca serviria para conversa que cresce mais de vinte blocos numa
rodada só: cada marca procura um prefixo guardado até vinte posições para
trás. Aqui cada resultado de ferramenta vai numa mensagem própria, então uma
rodada com mais de vinte chamadas paralelas perde a leitura daquela rodada e
escreve de novo. É raro, custa uma escrita, e resolver pede saber onde a
rodada anterior terminou — estado que o encoder, função pura da requisição,
não tem.

## Por que golden

O que se promete é o byte: uma marca um bloco fora do lugar guarda um prefixo
que ninguém reenvia, e o teste passaria contando marcas.
`internal/provider/testdata/anthropic-cache-request.json` é o corpo inteiro de
uma requisição uma rodada adentro.
