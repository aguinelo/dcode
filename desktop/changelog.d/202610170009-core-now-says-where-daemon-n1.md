## en

- **The core now says where the daemon is (N1).** `dcode socket` prints the
  daemon's socket path — one per user, whatever the environment — or fails with
  the reason, and the connection loop asks it instead of keeping a copy of the
  core's rule. `docs/DECISIONS.md` and `docs/loop/tasks.md` name the command.

## pt-BR

- **O núcleo agora diz onde está o daemon (N1).** `dcode socket` imprime o
  caminho do socket do daemon — um por usuário, qualquer que seja o ambiente —
  ou falha dizendo por quê, e o loop da conexão pergunta a ele em vez de guardar
  uma cópia da regra do núcleo. O `docs/DECISIONS.md` e o `docs/loop/tasks.md`
  nomeiam o comando.
