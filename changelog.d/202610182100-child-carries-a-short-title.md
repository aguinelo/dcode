## en

- **A child carries a short topic title.** The owner saw children listed by the first line of their
  task ("Crie no pacote `listas` (Go, módulo …"), which no row of a list can hold. A delegate
  item's `title` is now a topic title of at most 40 characters, such as "listas: Unicos". A longer,
  multi-line or missing one no longer refuses the batch: the harness derives a short one, always the
  same for the same input. It comes from the title's first line, or the task's when the title is
  missing, cut at the last word that fits. Delegate's answer says the title the child carries, the
  `dispatch` line keeps the asked one in `asked_title`, and the child's conversation is named by the
  short title. The tool's description of `title` changes with it.

## pt-BR

- **O filho leva um título curto de tópico.** O dono viu os filhos listados pela primeira linha da
  tarefa ("Crie no pacote `listas` (Go, módulo …"), que nenhuma linha de lista comporta. O `title`
  de um item de `delegate` passa a ser um título de tópico de até 40 caracteres, como "listas:
  Unicos". Um mais longo, de várias linhas ou ausente não recusa mais o lote: o harness deriva um
  curto, sempre o mesmo para a mesma entrada. Ele sai da primeira linha do título, ou da tarefa
  quando o título falta, cortado na última palavra que cabe. A resposta de `delegate` diz o título
  que o filho leva, a linha `dispatch` guarda o pedido em `asked_title`, e a conversa do filho
  recebe o título curto como nome. A descrição de `title` na ferramenta muda junto.
