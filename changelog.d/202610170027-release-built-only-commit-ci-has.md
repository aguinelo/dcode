## en

- **A release is built only on a commit CI has passed on `main`.** A new
  first job checks that the tag is on `main` and that CI passed at that exact
  commit, waiting while it still runs and failing, with the reason, when it
  ended red or never ran. With pull requests no longer running CI, a `main`
  commit can be red, and the release trusted that none was.

## pt-BR

- **Um release só é construído sobre um commit em que a CI passou na
  `main`.** Um primeiro job novo confere que a tag está na `main` e que a CI
  passou exatamente naquele commit, esperando enquanto ela ainda roda e
  reprovando, com o motivo, quando terminou vermelha ou nunca rodou. Com as
  pull requests sem CI, um commit da `main` pode estar vermelho, e o release
  confiava que nenhum estava.
