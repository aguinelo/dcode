# O veredito é da corrida

**Data:** 2026-09-03
**Specs afetadas:** `202608252000-loop-command` (`.p`, seção 8)
**Fonte:** relato do usuário — "não deixa claro quando começou e muito menos
quando encerrou (…) me parece que ele deixa passar algumas coisas de propósito
e justifica, o que abre margem para não cumprir os requisitos e parar precoce"

## O que mudou

Três coisas, e são um defeito só visto de três ângulos: **a corrida não dizia
como terminou.**

## 1. Corrida comum nunca anunciava o fim

O avanço estava atrás de `len(p.loopQueue) > 0`. Quando a **última** spec
terminava, a fila já estava vazia — então `nextSpec` não era chamada, e é ela
que produz o aviso de fim.

O único caminho que chegava a anunciar era o commit de uma proposta. Uma corrida
de specs que já declaram critérios terminava em silêncio total.

O relato diz "muito menos quando encerrou" e está literalmente certo.

## 2. O veredito era da última spec, não da corrida

`loopStanding` era sobrescrito por **todo** evento de turno concluído. Quando a
fila esvaziava, ele guardava o que a última spec tivesse dito.

Corrida cuja primeira spec deixou cobertura por cumprir e cuja segunda passou
limpa anunciava que **estava tudo cumprido**.

Ninguém mentiu: o número era verdadeiro sobre a última spec e foi impresso
debaixo de uma frase sobre a corrida. É o pior tipo de errado, porque quem lê
não tem como perceber, e as specs que falharam são exatamente as que ele mais
precisa ver nomeadas. É também, palavra por palavra, o "deixa passar algumas
coisas" do relato.

Agora os resultados são guardados **por spec**, os critérios são somados sobre
todas, e os nomes do que ficou são a união.

## 3. A primeira spec não contava

`loopWorked` só era incrementada em `nextSpec`, e a primeira spec é iniciada
direto pelo `specsFoundMsg`. Consequência dupla, e as duas chegavam à tela:

- corrida de **uma** spec terminava com `worked == 0`, e a regra "corrida que
  não trabalhou nada não anuncia fim" a calava;
- o registro do estado estava atrás de `loopWorked > 0`, então o estado da
  primeira spec nunca era guardado.

## O que a spec já exigia, e o teste que a satisfazia sem verificar

A RN-11 já dizia: *"o fim da corrida é dito, com quanto foi trabalhado e o
estado dos critérios, incluindo os nomes do que ficou por cumprir"*. E havia
teste nomeando essa invariante, então o `specguard` passava.

O teste injetava `loopFinishedMsg` **pronta** e afirmava que o aviso desenha o
que recebe. Isso mede a renderização e mais nada. **O que a corrida entrega**,
depois de várias specs com desfechos diferentes, não era afirmado em lugar
nenhum.

É a família de defeito que este repositório já nomeou uma vez — *"cada guarda
perguntava sobre um conjunto que já conhecia"* —, aqui numa forma nova: a
guarda perguntava sobre a metade barata do caminho. Invariante sobre a corrida,
teste sobre o desenho.

## Como o teste novo dirige

`runLoop` roda uma corrida de várias specs com desfechos diferentes e afirma o
que a tela diz no fim.

Ele grava e avança **direto**, em vez de pela `Update`, e o motivo está escrito
no teste: o próprio evento de conclusão deixa a sessão ociosa, então a `Update`
grava e avança na mesma passagem — inclusive o avanço final, cuja mensagem ela
devolve dentro de um lote que também espera no canal de eventos. Executar esse
lote para alcançar a mensagem trava num canal que nenhum fake alimenta.

Então a divisão é essa, e é deliberada: `TestAnIdleSessionWithAnEmptyQueueEndsTheRun`
afirma que o ramo **dispara**, pelo único efeito observável que ele deixa — a
corrida zerada —, e os outros afirmam **o que ela diz** quando dispara.

Os quatro foram verificados contra o defeito: com o portão antigo de volta, o
primeiro reprova; com o veredito lendo só a última spec, os outros reprovam.

## Não pôde ser conferido conta como pendente

`Unavailable` entra em "não terminou", junto de `Unmet`. Critério que ninguém
conseguiu medir não é critério que passou, e dobrá-lo em cumprido é anunciar
sucesso de trabalho que nada mediu — que é a mesma frase que o resto deste
changelog.
