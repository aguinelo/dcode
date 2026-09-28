# Cancelado não é cara ou coroa

**Data:** 2026-09-28
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, invariantes)
**Fonte:** `TestCancelClosesChannelWithCanceled` vermelho uma vez na CI (ubuntu,
run 36281823439), no #390 — um pull request sem Go nenhum

## O que mudou

Depois de cancelado o `ctx`, o pump não decodifica mais nada do que o
transporte ainda tinha: nem frame à espera, nem o fechamento. O caso do canal
do transporte começa perguntando ao contexto, e o `select` deixa de decidir
como um stream cancelado termina.

## O vermelho da CI não era isso

```
the stream ended with done / <nil>; want an EventError classed canceled
```

A primeira suspeita foi o `select` do pump: com o contexto cancelado e os frames
ainda no canal, o caso do canal poderia ganhar três vezes seguidas e decodificar
até o `[DONE]`. Só que os dois transportes daqui mandam cada frame num canal
**sem buffer** e fazem `select` com `ctx.Done()` do lado deles também. Depois do
cancelamento, nenhum frame atravessa. Medido: contexto cancelado antes de o
stream começar, 100.000 execuções, **zero** frames decodificados.

Um `done` no fim quer dizer, então, que o `[DONE]` atravessou **antes** do
cancelamento. O teste reproduzia uma resposta inteira e cancelava assim que
`Stream` voltava; numa máquina carregada, replay e pump terminavam antes de
`cancel()` rodar. Nesse entrelaçamento `done` é a resposta certa — ninguém tinha
cancelado nada.

| Sob carga (10 processos, `-race`, cobertura atômica, `-cpu=4`) | Falhas |
|---|---|
| teste antigo, pump antigo | 43 em 40.000 |
| teste antigo, pump consertado | 64 em 40.000 |
| teste novo, pump antigo | 0 em 40.000 |

A segunda linha é a que decide: nenhum conserto do produto cura um teste que
cancela um stream já terminado. É a corrida que o #113 tirou do teste vizinho, e
o mesmo erro de diagnóstico — o #112 consertou uma hipótese sobre o pump
enquanto quem mantinha a CI vermelha era o teste correndo contra o próprio
setup. O teste agora segura o transporte aberto depois dos frames e só cancela
quando a resposta visivelmente começou.

## O defeito que a investigação achou

A hipótese estava certa no mecanismo e errada no caminho. Com o contexto
cancelado, o canal do transporte fica pronto por outro motivo: **o
fechamento**, que os dois transportes fazem justamente porque viram o
cancelamento. O `select` sorteia entre ele e `ctx.Done()`, e o ramo do canal
fechado dava ao decodificador a última palavra.

Depois de um frame que termina a resposta mas não traz o uso — MiniMax e OpenAI
mandam o uso num frame próprio, depois — o decodificador responde ao fechamento
com `done`. Todo stream desses dialetos passa por esse estado. Uma interrupção
que chegasse com o pump ocupado com esse frame terminava em `done` metade das
vezes: **49.948 em 100.000**.

Um transporte que guarde frames num buffer — a interface não proíbe — deixava o
`select` decodificar o que estivesse esperando: com os três frames do teste,
**12,6%** terminavam em `done`.

## Por que `done` não serve

`Decide` manda `canceled` para o silêncio. Um stream `done` é uma resposta que o
laço grava no histórico e executa: as chamadas de ferramenta seguem para
`execute` sem que ninguém olhe o contexto de novo, e `Write` e `Edit` ignoram o
contexto que recebem.

## A regra, e onde ela fica

`ctx.Err()` no topo do caso do canal — exatamente onde a moeda cai. Vale para
frame e para fechamento, então as duas metades do sorteio saem pela mesma porta.

A checagem que o ramo do canal fechado tinha desde agosto foi absorvida por
esta. Um fechamento causado pelo cancelamento sempre acontece depois dele — o
transporte só fecha porque viu `ctx.Done()` —, então a checagem de cima vê o
cancelamento primeiro. O que ela não cobre é um fechamento **independente** com
um cancelamento chegando no intervalo de uma chamada a `dec.Close()`: aí o
transporte de fato terminou antes de o pump ver cancelamento algum, e o stream
diz isso. `classify` continua com a sua regra de contexto, testada direto nela.

## A fronteira que fica

Todo cancelamento tem um último instante em que pode ser visto, e este é o do
pump: um frame que ele já tinha tomado antes do cancelamento é decodificado, e se
era o terminal, o stream termina `done`. Garantia depois desse instante é
trabalho de quem cancelou. O laço olha o contexto no topo de cada iteração — mas
ainda não entre o fim do stream e a execução das ferramentas.

## Como fica preso

`TestACancelledStreamEndsCanceledWhateverTheSelectPicks` não deixa nada para o
escalonador: o primeiro frame é tomado com o contexto vivo, um `Decoder` dublê
cancela enquanto o decodifica, e o que resta já está no canal. Sobra só a
escolha que a linguagem especifica como uniforme. Antes do conserto, cada
execução falhava metade das vezes — 400 de 400 subtestes vermelhos em 200
repetições, com e sem `-race`, de `-cpu=1` a `-cpu=8`. Cem execuções passando
por sorte é um evento de 2⁻¹⁰⁰.
