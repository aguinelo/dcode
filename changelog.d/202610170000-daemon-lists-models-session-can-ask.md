## en

- **The daemon lists the models a session can ask for.** `GET /v1/models`
  answers, for a workspace, the model a session there gets when it asks for
  none and every profile in `models.toml` — the user's, with the project's
  layered over them by name — each resolved the way the session asking for it
  resolves it: model, family, transport, endpoint, window, and whether the
  family has measurements behind it, with the family's own warning when it has
  none. A profile no session could be built with stays listed and says why. No
  answer carries the key, its mask or its fingerprint. Before, a client learned
  which models existed only by reading `models.toml` itself, and could not tell
  a measured family from one nobody measured. This is what the desktop's
  per-session model switch reads.

## pt-BR

- **O daemon lista os modelos que uma sessão pode pedir.** `GET /v1/models`
  responde, para um workspace, o modelo que uma sessão ali recebe sem pedir
  nenhum e cada perfil do `models.toml` — o do usuário, com o do projeto por cima
  pelo nome —, cada um resolvido como a sessão que o pedir o resolve: modelo,
  família, transporte, endpoint, janela e se a família tem medição por trás, com
  o aviso da própria família quando não tem. Um perfil com que nenhuma sessão
  pode ser montada continua na lista e diz por quê. Nenhuma resposta carrega a
  chave, a máscara ou a impressão digital dela. Antes, um cliente só sabia quais
  modelos existiam lendo o `models.toml` por conta própria, e não distinguia uma
  família medida de uma que ninguém mediu. É o que a troca de modelo por sessão
  do desktop lê.
