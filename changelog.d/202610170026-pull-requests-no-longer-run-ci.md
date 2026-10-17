## en

- **Pull requests no longer run CI; `main` does, after the merge.** The
  pre-push hook already runs the same gate here, on the pushed commit, in
  seconds, and paying GitHub minutes to find again what a local run found is
  waste. CI still runs on every push to `main`, the only place Linux and its
  bubblewrap sandbox are exercised, and on a branch on demand.

## pt-BR

- **Pull requests não rodam mais CI; a `main` roda, depois do merge.** O hook
  de pre-push já roda aqui o mesmo gate, no commit enviado, em segundos, e
  pagar minutos do GitHub para achar de novo o que a execução local já achou é
  desperdício. A CI continua rodando em todo push na `main`, único lugar onde o
  Linux e o sandbox bubblewrap são exercitados, e numa branch quando pedida.
