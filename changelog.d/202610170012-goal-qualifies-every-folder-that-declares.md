## en

- **A goal qualifies every folder that declares nothing before any work.** In
  a `/loop <goal>`, each pending folder with no criteria now gets a qualifying
  session first, all in one pass, and the run stops with the proposals written
  for one review: the daemon waits in `awaiting_review` and works every pending
  folder on `continue`, and the TUI says how many it wrote and that the goal
  runs again after the review. It used to open a work session on such a
  folder, which reports done after one turn measured against nothing. The
  owner's decision, per §3.3 of the loop proposal; daemon and TUI change
  together so the two clients run the same loop.

## pt-BR

- **Um objetivo qualifica toda pasta que não declara nada antes de qualquer
  trabalho.** Num `/loop <objetivo>`, cada pasta pendente sem critério ganha
  antes uma sessão qualificadora, todas numa passada, e o loop para com as
  propostas escritas para uma revisão só: o daemon espera em
  `awaiting_review` e trabalha toda pendente no `continue`, e a TUI diz quantas
  escreveu e que o objetivo roda de novo depois da revisão. Antes ela abria
  sessão de trabalho numa pasta assim, que relata pronto depois de um turno
  medido contra nada. Decisão do dono, pela §3.3 da proposta do loop; daemon e
  TUI mudam juntos para os dois clientes rodarem o mesmo loop.
