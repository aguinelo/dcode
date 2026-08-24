# Marcador que ainda está chegando

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** quadro real, no meio de um turno

## O que mudou

Marcador aberto no **fim** do texto e ainda sem par não é desenhado.

## O que se via

No meio de um turno, a última linha do fluxo era:

```
1. **
```

Toda palavra enfatizada chega como `**` primeiro e o par dela alguns deltas
depois. Então a tela piscava um par de asteriscos antes de cada título que o
modelo escrevia — e como o fluxo está ancorado no fim, era a última coisa que o
leitor via.

## Duas condições, e as duas são necessárias

**No fim.** `**negrito` com palavras depois é um marcador que alguém escreveu e
deixou; apagar isso apagaria algo escrito de propósito, que é a regra que este
arquivo já enunciava.

**Sem par.** A primeira versão olhava só o fim, e comeu o fechamento de todo par
completo: `1. **Alvo**` virou `1. **Alvo`. A contagem é o que responde — texto
que termina em `**` porque um par fechou ali é um par pronto.

Achado renderizando os dois casos lado a lado, e não pensando neles. É a mesma
diferença que separa as correções de hoje das dos dias anteriores.
