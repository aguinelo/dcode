# Corte que se anuncia

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** sessão real reproduzida pelo redutor — `1a030dd642cd50b9f68`

## O que mudou

O painel e a coluna cortam com marca. A regra já estava escrita nesta família,
em `rail.go`, para título de conversa:

> Cortado com marca, nunca em silêncio. Um título que apenas para deixa o leitor
> sem saber distinguir uma conversa curta de uma truncada.

O painel respondia a mesma pergunta do outro jeito. Na sessão real:

```
✓ 6 CLI sob demanda com contr
```

Acabava ali. E a coluna, que enuncia a regra, não a aplicava ao **nome de
arquivo** — só ao título de conversa. `client.py` e `client.pyi` são arquivos
diferentes, e um dos dois não existe.

Uma regra com uma exceção tem mais de uma. As duas colunas seguem a mesma agora.

## O fim é que cede

Item de plano e nome de arquivo se identificam pelo começo, como comando. É a
mesma decisão de `elide`, e pela mesma razão — só que aqui o diretório já está
na linha de cima, então o que sobra na linha é o nome.

## Elidir antes de estilizar

A `Palette` documenta o contrato: *"todo chamador mede células antes de
estilizar"*. O `turnSection` estilizava e depois cortava, e o `clip` mede com
`runewidth`, que conta os bytes imprimíveis de uma sequência de escape como seis
células de texto.

**Este é latente, não visível.** Nas larguras que o painel realmente assume — 25
a 34 colunas — nenhuma linha da seção TURNO chega perto do teto, então as seis
células nunca custaram nada na tela. Ficaria a uma tradução mais longa de
distância de custar, e `runewidth.Truncate` cortaria dentro do escape, deixando
o terminal naquela cor até o fim da tela.

Está dito aqui como latente porque a alternativa seria escrever no changelog que
consertei algo que ninguém viu — e é justamente a diferença entre as duas
afirmações que esta família precisa manter honesta.

## O teste é sobre a classe

`TestColourNeverChangesWhatIsOnTheScreen` desenha o quadro inteiro com e sem
cor, tira os escapes de um e compara com o outro, em quatro larguras. Uma regra
com um contraexemplo tem outros, e teste por chamada só encontra a chamada de
que já se desconfiava.
