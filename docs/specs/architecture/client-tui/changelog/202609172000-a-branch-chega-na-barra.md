# A branch chega na barra

**Data:** 2026-09-17
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção sobre a barra inferior)
**Fonte:** pedido do usuário — "precisamos também refletir a branch atual do
projeto" — confirmado como "mostrar a branch git na interface" (não no prompt
do modelo, que já a tinha).

## O que já existia e nunca chegava aqui

`internal/vcs/git.go` já lê a branch atual desde a mudança documentada em
`202608170200-onde-o-agente-esta.md`, da família `behavior-definition` — mas
só pra montar a seção "Current branch" do prompt. O valor lido morria numa
variável local de `app.New`, nunca saía dali: não em `app.Session`, não em
`session.Session`, não em `protocol.Session`. O modelo sabia em qual branch
estava; a pessoa olhando a tela, não — a mesma leitura já tinha sido lida,
só nunca tinha pra onde ir.

## O que mudou

`app.Session` ganhou `Branch string`, preenchido a partir do mesmo
`vcs.Read(...)` que já roda em `New` — sem segunda leitura, sem I/O extra.
`Daemon.build` copia `appSession.Branch` pra `sess.Branch`
(`internal/app/daemon.go`), ao lado do que já faz pra `Family`/`Transport`/
`BaseURL`. `session.Session.Describe()` inclui `Branch` no
`protocol.Session` — então `session.created` passa a carregar o fato, e
`internal/tui`'s `Model.Apply` já sabia ler tudo desse evento; só precisou
de mais um campo.

A barra desenha num segmento novo, `branchSegment`, ao lado do worktree —
vazio não desenha nada (sem repositório, git ausente, ou HEAD destacado são
todos "nada a dizer"), e cede espaço antes do modo quando o terminal aperta.
Cede, mas não pro **caminho** do workspace: a competição entre segmento e
caminho (`pathOutranks`) tratava tudo droppable como equivalente, e dar à
branch a mesma prioridade que as dicas de tecla teria feito o caminho
completo do workspace — que o worktree já nomeia a cauda — vencer contra
"em qual branch estou", uma informação bem mais rara de reconstruir de
cabeça.

## Por que não ler git direto no cliente

`internal/tui` nunca lê disco nem executa `git` — é o pacote que desenha,
não o que descobre fatos. Ler a branch ali quebraria isso duas vezes: uma
leitura de git fora do daemon, e uma segunda leitura da mesma informação já
lida uma vez na construção da sessão — exatamente o que `202608170200`
rejeitou para o prompt ("duas leituras de git numa sessão só podem
discordar"), agora valendo pra tela também.
