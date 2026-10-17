## en

- **The daemon lists a workspace's user commands.** `GET /v1/commands` answers
  the slash commands the user and the project wrote in `commands/*.md`, found
  by the same `config.DiscoverCommands` and the same 64 KiB cap the TUI uses,
  the project's winning by name and named in `overrides`, each with its source,
  its file and the body it sends. Only the TUI could offer them, because it
  reads the files itself; the desktop talks to the daemon over HTTP and cannot
  read the user's config root. The body travels, unlike a skill's, because
  invoking a command is sending its text, and a command never runs anything. A
  file that cannot load refuses the list with the reason, as it refuses the
  TUI's start. The Go client gains `ListCommands`.

## pt-BR

- **O daemon lista os comandos do usuário num workspace.** `GET /v1/commands`
  responde os comandos de barra que o usuário e o projeto escreveram em
  `commands/*.md`, achados pelo mesmo `config.DiscoverCommands` e pelo mesmo
  teto de 64 KiB da TUI, o do projeto vencendo pelo nome e citado em
  `overrides`, cada um com a origem, o arquivo e o corpo que manda. Só a TUI
  conseguia oferecê-los, porque lê os arquivos sozinha; o desktop fala com o
  daemon por HTTP e não lê a pasta de configuração do usuário. O corpo viaja, ao
  contrário do de uma skill, porque invocar um comando é mandar o texto dele, e
  um comando nunca roda nada. Um arquivo que não carrega recusa a lista com o
  motivo, como recusa a partida da TUI. O cliente Go ganha `ListCommands`.
