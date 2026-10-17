## en

- **Changes are recorded as fragments, as in the core.** Each change is one
  new file, `desktop/changelog.d/<yyyymmddhhmm>-<slug>.md`, with both
  editions; `Unreleased` is assembled from them, and the first `desktop-v*`
  release folds them with `scripts/changelog-assemble.sh release X.Y.Z
  desktop`. The desktop's pull requests conflicted on the top of this file the
  same way the core's did. The 75 entries of `Unreleased` moved into fragments
  unchanged.

## pt-BR

- **Mudanças são registradas como fragmentos, como no núcleo.** Cada mudança é
  um arquivo novo, `desktop/changelog.d/<yyyymmddhhmm>-<slug>.md`, com as duas
  edições; o `Não publicado` é montado a partir deles, e o primeiro release
  `desktop-v*` os dobra com `scripts/changelog-assemble.sh release X.Y.Z
  desktop`. Os PRs do desktop conflitavam no topo deste arquivo do mesmo jeito
  que os do núcleo. As 75 entradas do `Não publicado` foram para fragmentos sem
  mudança.
