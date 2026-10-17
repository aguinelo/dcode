## en

- **A session the window opens reads its project's configuration (N2).** The
  daemon now resolves the configuration chain in each session's workspace, so a
  session opened in a project runs under that project's `.dcode/config.toml`,
  even with the daemon started outside every project, as the desktop starts it.
  A project configuration that cannot be read refuses the session with the
  reason. `docs/DECISIONS.md` marks N2 done.

## pt-BR

- **Uma sessão aberta pela janela lê a configuração do seu projeto (N2).** O
  daemon agora resolve a cadeia de configuração no workspace de cada sessão, então
  uma sessão aberta num projeto roda sob o `.dcode/config.toml` daquele projeto,
  mesmo com o daemon subindo fora de todos eles, como o desktop o sobe.
  Configuração de projeto que não se lê recusa a sessão com o motivo. O
  `docs/DECISIONS.md` marca o N2 como feito.
