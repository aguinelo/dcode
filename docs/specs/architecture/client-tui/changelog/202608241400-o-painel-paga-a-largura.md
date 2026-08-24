# O painel paga a largura que ocupa

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** medição sobre a sessão real `1a030dd642cd50b9f68`

## O que mudou

Duas coisas, uma pergunta: o painel está pagando o que ocupa?

## Ele chegava devendo vinte e cinco colunas

`panelWidth` era um quarto da tela. No limiar, isso significa que ele aparece
**já devendo** um quarto: cruzar de 99 para 100 colunas custava vinte e cinco de
uma vez.

Agora ele aparece no **piso** e cresce, pago do que sobra além da largura em que
passou a ser permitido — um terço do excedente. O degrau cai de vinte e cinco
para dezesseis, e cada coluna depois disso é dividida em vez de tomada.

O degrau que resta é **negociação, não defeito**: o leitor abre mão de colunas
de texto e recebe o plano. O que era defeito era o tamanho — dois limiares no
mesmo cem, um deles comprando uma coluna que repetia o que o fluxo tinha acabado
de dizer, somando 46 colunas na diferença de uma.

Tirá-lo de vez significaria nunca mostrar o painel ou sempre mostrá-lo. O
invariante afirma o que dá para afirmar: **o fluxo perde largura uma vez só, e
nunca mais que o piso do painel.**

## `iteração 0/2000`

A seção TURNO existe para avisar que um teto vem chegando. Era desenhada desde o
primeiro evento de toda sessão, e na sessão real gastava trinta e três colunas
para dizer `iteração 0/2000` e `em vôo 0·4`.

Zero de dois mil não avisa de nada. É número que ninguém age sobre, na parte
mais cara da tela — e informação que ninguém age sobre é o que ensina as pessoas
a parar de ler a região onde ela está.

Agora aparece a partir de **metade** do teto, e sempre que **todos os lugares em
vôo** estão ocupados.

Metade, e não os três quartos em que o estilo já muda: aviso que chega em três
quartos chega depois de o turno ter se comprometido com a abordagem. E todos os
lugares ocupados não é um teto se aproximando — é um teto alcançado agora.

## O efeito somado

Sem plano e longe do teto, o painel não abre. Com a coluna também escondida por
padrão, uma sessão nos primeiros minutos entrega o terminal inteiro à conversa,
que é onde o trabalho está.
