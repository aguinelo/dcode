## en

- **A loop stopped while it writes a proposal still says where the file is,
  and a loop on `/` holds every tree.** A stop now that landed during the
  commit left a `done.toml` on disk and no proposal on the run; the path is now
  recorded before the stop is honoured. And the refusal of a loop inside or
  around a live one compared `/` as `//`, so a loop on the root refused
  nothing. The loop-command spec also states that a goal stops at the first
  folder whose turn ends at a ceiling, where the TUI goes on — on purpose.

## pt-BR

- **Um loop parado enquanto grava a proposta ainda diz onde o arquivo está, e
  um loop em `/` contém toda árvore.** Uma parada que chegava durante a
  gravação deixava um `done.toml` no disco e nenhuma proposta no loop; agora o
  caminho é registrado antes de a parada ser honrada. E a recusa de um loop
  dentro ou em volta de outro vivo comparava `/` como `//`, então um loop na
  raiz não recusava nada. A spec da loop-command também diz que um objetivo
  para na primeira pasta cujo turno termina num teto, onde a TUI segue — de
  propósito.
