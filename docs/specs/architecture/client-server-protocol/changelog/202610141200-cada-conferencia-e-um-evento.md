# Cada conferência dentro do turno é um evento

**Data:** 2026-10-14
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: uma linha em
§5.1, quatro linhas novas em §9)
**Fonte:** a proposta do `/loop` no desktop
(`desktop/docs/proposals/loop.md`, §4.2), decisão Q4 A, aprovada pelo dono em
2026-10-13; é o passo 3 da entrega (§8) e parte do pedido N4 (D24 do desktop).

## O problema

O turno não termina porque o modelo parou de pedir ferramentas: ele confere a
definição de pronto e re-entra enquanto os critérios não cumpridos encolhem
(`agent-loop` RN-10). Cada uma dessas conferências acontecia em silêncio. O
cliente só sabia onde os critérios estavam no `turn.completed`, e "ciclo 3, 2 de
5 critérios, parado 1 de 3" não existia no fio. O desktop desenha as luzes dos
critérios de um turno que já terminou, e não tinha como movê-las enquanto ele
roda.

## O que mudou

- **`done.checked`**, no log da sessão, cada vez que `checkDone` roda os
  critérios: o turno, o número da conferência dentro dele, os critérios que
  passaram, os que não passaram e os que não puderam rodar, para onde a
  conferência andou (`moved`), o contador sem avanço e o teto dele, o que o
  laço desfez e o protegido que foi escrito.
- `moved` é um conjunto fechado: `first`, `forward`, `stalled`, `backward`.

## Como, e por quê assim

- **O que o laço decidiu, não uma segunda leitura.** `moved` e `stall` são os
  valores sobre os quais `checkDone` age, lidos de volta no mesmo lugar. Um
  cliente que recalculasse o movimento a partir das listas seria uma segunda
  cópia da regra, e a cópia é o que deriva.
- **`first` e não `forward` na primeira conferência.** A proposta listava três
  valores. Na primeira não há nada antes para comparar, e o laço também não
  compara — zera o contador. Chamar isso de avanço afirmaria um progresso que
  ninguém fez; `first` diz o que aconteceu.
- **`stalled` e não `nowhere`.** O laço chama o empate de `MovedNowhere` por
  dentro; no fio vale o que ele conta, e ele conta o empate como conferência
  parada. É também a palavra que o cliente mostra ("parado 1 de 3").
- **O teto viaja com o contador.** `stall_limit` vai junto de `stall`, como o
  `progress` leva o teto das rodadas: o cliente nunca guarda uma cópia de
  `MaxStallCycles`.
- **Nomes, nunca contagem de testes.** Contar testes exigiria ler a saída do
  comando, o que a decisão Q4 recusou (a opção C, a do `LOOP.md`) e o D24 já
  tinha recusado.
- **Fora do contexto do modelo.** O modelo já recebe os mesmos fatos como
  lembrete, nas palavras dele; o evento é a janela da pessoa, como `progress`
  (ADR-03).
- **A conferência que encerra o turno também sai.** Ela é anunciada antes de o
  turno decidir terminar, então o último `done.checked` de um turno que acabou
  `done` mostra todos os critérios passando, antes do `turn.completed`.
- **Turno que não escreveu nada não confere e não emite** — é o que `checkDone`
  já fazia, e continua.

## Compatibilidade

Evento novo num protocolo `experimental`: MINOR. Cliente que ignora tipos que
não conhece vê a sequência que via antes, mais eventos que ele pula. O desktop
passa a ler `done.checked` num PR próprio (passo 6 da proposta).
