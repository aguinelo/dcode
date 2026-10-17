## en

- **The daemon reads a workspace's git as it is now.** `GET /v1/git` answers
  where a folder is in git when someone looks — the top of its work tree, the
  branch now (not the one the session started on), the commit, and whether it
  is a linked worktree and of which main checkout — and every file changed
  since the last commit, staged or not, with its status and its lines.
  `GET /v1/git/diff` gives one of those files' unified diff, cut at a line past
  256 KiB and saying so. Only a file git lists as changed has a diff, so the
  route never reads a file by name, and every read runs with
  `core.fsmonitor` off, so a repository's configuration cannot make looking
  at it run a program. A plain folder is said as not a
  repository, and git missing or failing is a refusal with the reason — never
  "no changes". Before, a client knew only the branch frozen at session start
  and what the tool events said they wrote; this is what the desktop's context
  panel reads.

## pt-BR

- **O daemon lê o git de um workspace como está agora.** `GET /v1/git`
  responde onde a pasta está no git quando alguém olha — o topo da árvore, a
  branch de agora (não a do início da sessão), o commit, e se é uma worktree
  ligada e de qual checkout principal — e cada arquivo mudado desde o último
  commit, preparado ou não, com o estado e as linhas. `GET /v1/git/diff` dá o
  diff unificado de um desses arquivos, cortado numa linha depois de 256 KiB e
  dizendo que foi. Só arquivo que o git lista como mudado tem diff, então a rota
  nunca lê um arquivo pelo nome, e toda leitura roda com o `core.fsmonitor`
  desligado, para a configuração de um repositório não fazer olhar para ele
  rodar um programa. Pasta comum é dita como fora de repositório, e
  git ausente ou falhando é recusa com o motivo — nunca "nada mudou". Antes, o
  cliente sabia só a branch congelada no início da sessão e o que os eventos
  das ferramentas diziam ter escrito; é isto que o painel de contexto do
  desktop lê.
