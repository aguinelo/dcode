# Merged sai da lista

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §5.1, §11, invariantes; `.i` passo 6).

## O que aconteceu

`Children()` tirava da resposta de `/children` só o filho `closed` cujo `Ended` tivesse passado de
`orchestrator.closed_retention`. O `merged` também é terminal e também recebe `Ended` (o `move`
de `merging.go` o carimba na mesma transição, sob o lock do commit de #549), mas ficava na lista
para sempre — e como `Close` recusa um filho terminal, não havia caminho que o tirasse. A `.p` §11
dizia "Fechado sai", e o merged ficou de fora da regra sem que ninguém decidisse isso.

## O que mudou

- A regra da Q-E — "um agente fechado sai da lista uma hora depois; o histórico continua no log" —
  vale para os dois terminais: `closed` e `merged` saem `closed_retention` depois de `Ended`.
- `failed` continua: não é terminal, fica até a pessoa agir. `Close` de um falho já era aceito, e
  o fechado que resulta sai como qualquer outro.
- `.p` §5.1 e §11 dizem isso, e §13 ganha a linha do merged, com
  `TestAMergedChildLeavesTheListAfterTheRetentionAndAFailedOneStays`, que avança o relógio
  injetado em vez de dormir.

## Por que não foi preciso mexer no merge

O `Ended` do merged já era gravado: `move` o põe em toda transição para um estado terminal, dentro
do `commitLive`. A lacuna era só do filtro.
