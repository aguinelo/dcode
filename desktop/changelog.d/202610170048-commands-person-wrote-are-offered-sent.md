## en

- **The commands the person wrote are offered and sent as in the TUI.** The
  TUI offers the slash commands in `commands/*.md`, the user's and the
  project's; the window could not read them. It now asks the core
  (`GET /v1/commands`) once per project it has a conversation of, as the TUI
  reads them once at start, and the `/` menu lists them after the field's own,
  with their description and whose they are. Typed, one is expanded as the
  TUI's `config.Expand` does (`$ARGUMENTS`, or what followed appended) and sent
  as a message; a file named like a field command never shadows it. The help
  lists them under "Deste projeto". A list that did not come is said, with the
  daemon's reason, at the foot of the menu, in the help and by an unknown
  `/name`; never an empty list. Nothing at rest changes: both visual references
  measure 0.00% (D50).

## pt-BR

- **Os comandos que a pessoa escreveu aparecem e vão como na TUI.** A TUI
  oferece os comandos de barra de `commands/*.md`, os do usuário e os do
  projeto; a janela não conseguia lê-los. Agora ela pergunta ao núcleo
  (`GET /v1/commands`) uma vez por projeto de que tem conversa, como a TUI lê
  uma vez ao abrir, e o menu de `/` os lista depois dos do campo, com a
  descrição e de quem são. Digitado, um comando é expandido como o
  `config.Expand` da TUI expande (`$ARGUMENTS`, ou o resto acrescentado) e vai
  como mensagem; um arquivo com nome de comando do campo nunca o encobre. A
  ajuda os lista em "Deste projeto". Uma lista que não veio é dita, com o motivo
  do daemon, no pé do menu, na ajuda e num `/nome` desconhecido; nunca uma lista
  vazia. Nada em repouso muda: as duas referências visuais medem 0,00% (D50).
