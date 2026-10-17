## en

- **Each check of the definition of done inside a turn is an event.** The loop
  re-enters a turn while the unmet criteria shrink, and every one of those
  checks ran in silence: a client learned where the criteria stood only at
  `turn.completed`, so "cycle 3, two of five, stalled 1 of 3" was not on the
  wire. `done.checked` now carries, for every check, the cycle within the turn,
  the criteria met, unmet and unable to run, where the check moved (`first`,
  `forward`, `stalled`, `backward`), the stall count against its ceiling, what
  the loop undid and the protected paths written. `moved` and `stall` are the
  values the loop acts on, read back rather than decided twice; names, never
  test counts, because counting passes would mean parsing a command's output. A
  turn that wrote nothing still runs no check and emits nothing. The desktop
  reads the event and does not show it yet.

## pt-BR

- **Cada conferência da definição de pronto dentro do turno é um evento.** O
  laço re-entra no turno enquanto os critérios não cumpridos encolhem, e cada
  uma dessas conferências corria em silêncio: o cliente só sabia onde os
  critérios estavam no `turn.completed`, e "ciclo 3, dois de cinco, parado 1 de
  3" não existia no fio. `done.checked` agora leva, a cada conferência, o ciclo
  dentro do turno, os critérios que passaram, os que não passaram e os que não
  puderam rodar, para onde a conferência andou (`first`, `forward`, `stalled`,
  `backward`), o contador sem avanço contra o teto, o que o laço desfez e o
  protegido escrito. `moved` e `stall` são os valores sobre os quais o laço age,
  lidos de volta em vez de decididos duas vezes; nomes, nunca contagem de
  testes, porque contar o que passou exigiria ler a saída do comando. Turno que
  não escreveu nada continua sem conferir e sem emitir. O desktop lê o evento e
  ainda não o mostra.
