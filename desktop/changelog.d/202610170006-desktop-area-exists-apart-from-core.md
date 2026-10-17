## en

- **The desktop area exists, apart from the core.** Electron Forge with Vite,
  TypeScript and React, in `desktop/`, with its own README, changelog, version
  and `AGENTS.md`, and a stub `go.mod` so the core's `go … ./...` stops at the
  door instead of walking into `node_modules`. Separate because the desktop
  depends on the core and never the other way round, and a client that ships on
  its own clock should not borrow the core's version.

## pt-BR

- **A área do desktop existe, à parte do núcleo.** Electron Forge com Vite,
  TypeScript e React, em `desktop/`, com README, changelog, versão e `AGENTS.md`
  próprios, e um `go.mod` de fachada para o `go … ./...` do núcleo parar na porta
  em vez de entrar no `node_modules`. À parte porque o desktop depende do núcleo e
  nunca o contrário, e um cliente que sai no próprio ritmo não deve pegar
  emprestada a versão do núcleo.
