## en

- **The loop list finds the loop that blocks a workspace.** `GET /loops?workspace=`
  now keeps the runs in, inside or around that tree, links resolved — the rule
  that refuses a second loop — and refuses a workspace it cannot read with
  `workspace_invalid`. It matched the exact path only, so a `/loop` typed again
  from a subfolder, or through a link, found nothing to continue and was
  refused with `loop_already_running`.

## pt-BR

- **A lista de loops acha o loop que bloqueia um workspace.** `GET
  /loops?workspace=` agora guarda os loops na árvore, dentro ou em volta dela,
  com os links resolvidos — a regra que recusa um segundo loop — e recusa com
  `workspace_invalid` o workspace que não pode ler. Comparava só o caminho
  exato, e um `/loop` digitado de novo numa subpasta, ou por um link, não achava
  o que seguir e era recusado com `loop_already_running`.
