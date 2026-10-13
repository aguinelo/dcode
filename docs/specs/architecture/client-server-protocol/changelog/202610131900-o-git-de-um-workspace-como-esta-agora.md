# O git de um workspace, como está agora

**Data:** 2026-10-13
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: duas rotas em
§4, a §5.5, onze linhas novas em §9)
**Fonte:** o relato de uso do cliente desktop, em 2026-10-13: "não fica claro
em que sessão estou, pasta de projeto, branch, worktree e diff, tá bem cru".

## O problema

O desktop mostra, de uma conversa, o título e o nome curto da pasta. A branch
que ele tem é a `Session.Branch`, lida uma vez na construção da sessão e nunca
de novo — de propósito, porque é fato do prompt. Uma conversa que troca de
branch no meio continua dizendo a primeira. Se a pasta é uma worktree ligada, e
de qual checkout, o protocolo não diz. E o único diff que o cliente tem é o que
as chamadas de ferramenta dizem que escreveram: não vê o que a pessoa mudou à
mão, o que um `bash` mudou, nem o diff de um arquivo.

## O que mudou

- **`GET /v1/git?workspace=`** responde onde a pasta está no git agora: topo da
  árvore, branch (ou HEAD destacado), commit, se é worktree ligada e qual o
  checkout principal; e cada arquivo mudado desde o último commit, com o estado
  e as linhas acrescentadas e tiradas.
- **`GET /v1/git/diff?workspace=&path=`** responde o diff unificado de um
  arquivo dessa lista, cortado numa linha depois de 256 KiB e dizendo que foi.
- **`vcs.ReadTree` e `vcs.FileDiff`**, ao lado do `vcs.Read`, que continua o
  retrato do prompt.
- **O cliente Go** ganha `ReadGit` e `ReadGitDiff`.

## Como, e por quê assim

- **No daemon, e não no processo principal do Electron.** O desktop poderia
  rodar o git ele mesmo, com `execFile` e a pasta validada contra a lista do
  daemon, como o terminal (D37 do desktop) faz. Ficou no daemon porque o daemon
  já é quem lê git (`internal/vcs`), com o ambiente fixado (`LC_ALL=C`,
  `GIT_OPTIONAL_LOCKS=0`, sem pager) e testado contra repositório de verdade;
  porque o daemon pode estar em outra máquina que não a da janela, e só ele
  enxerga a pasta; e porque uma segunda leitura de git no cliente seria a
  segunda cópia das mesmas regras — branch "HEAD" quando destacado, repositório
  sem commit, worktree — que deixaria de seguir a primeira no dia em que
  mudasse.
- **Por workspace, não por sessão.** `?workspace=`, como `/memory` e `/skills`,
  e não `/sessions/{id}/git`: uma conversa terminada não tem sessão viva, e é
  exatamente quando alguém quer ver o que ficou na pasta. O cliente pergunta
  pela pasta que a lista de conversas dá a ela.
- **A branch de agora, sem mexer na `Session.Branch`.** As duas dizem coisas
  diferentes e as duas continuam: uma é onde a sessão começou e o que o prompt
  dela leu; a outra é onde a pasta está quando alguém olha.
- **A pasta, não a conversa.** A lista é tudo o que o git dá como mudado desde
  o último commit, preparado ou não. Duas conversas na mesma pasta veem a mesma
  lista, e uma edição à mão aparece. O que uma conversa escreveu continua vindo
  dos eventos dela, no cliente; o cliente diz qual das duas mostra.
- **Contra o último commit, preparado e não preparado juntos.** É o que a pasta
  tem que o commit não tem, e o que uma pessoa quer ver antes de commitar.
  Separar o preparado é coisa de quem vai commitar, e quem commita é a pessoa,
  no git dela.
- **Diff só de arquivo listado.** O caminho pedido tem de estar, exatamente,
  na lista que `/git` daria naquela hora. Sem isso a rota leria qualquer arquivo
  pelo nome — `../`, absoluto, ou um pathspec mágico do git. O diff roda com
  `--literal-pathspecs` mesmo assim.
- **Arquivo novo é diff contra nada** (`--no-index /dev/null`), e as linhas
  dele são contadas lendo o arquivo, com a mesma regra de binário do git (NUL
  nos primeiros 8000 bytes). Link e arquivo especial não são lidos.
- **Falha dita, nunca lista vazia.** Pasta fora de repositório é
  `repository: false`, um estado comum e dito. Git ausente, git que falha e git
  que não responde em 5 s são recusa com o motivo — nenhum deles é "não é
  repositório" nem "nada mudou".
- **Tetos:** 1000 arquivos na lista (`truncated`), 256 KiB num diff (`limit`).
  Um checkout com milhares de arquivos gerados é dito cortado em vez de
  atravessar o socket inteiro.
- **Pasta que não existe é recusada** antes do git rodar: sem isso, o git
  rodaria onde o processo caísse.
- **Nenhum programa do repositório roda.** A configuração de um repositório
  pode pôr um comando em `core.fsmonitor`, e o `git status` o executa. A rota lê
  qualquer pasta em que uma conversa roda, um repositório clonado inclusive;
  olhar o painel não pode ser o jeito de esse programa rodar. Toda leitura passa
  `-c core.fsmonitor=false`. O diff já roda sem diff externo e sem `textconv`.

## As invariantes

As onze linhas novas em §9 são reivindicadas onde cada metade está — a leitura
em `internal/vcs`, contra repositórios de verdade, e as rotas em
`internal/server`: `TestTheGitRouteListsTheTreeAsItIsNow`,
`TestALinkedWorktreeNamesItsMainCheckout`, `TestADetachedHeadIsNotABranch`,
`TestARepositoryWithNoCommitListsWhatItHolds`,
`TestABinaryFileIsMarkedAndNotCounted`,
`TestAFolderThatIsNotARepositoryIsSaidAsSuch`, `TestWithoutGitTheRouteSaysSo`,
`TestALongDiffIsCutAndSaysSo`, `TestTheDiffRouteRefusesAFileThatDidNotChange`,
`TestTheGitRoutesRefuseAWorkspaceTheyCannotRead` e
`TestReadingTheTreeNeverRunsTheRepositorysFsmonitor`.

Rotas e tipos novos, nenhum contrato existente mudado: MINOR.

## O que fica de fora

- **O painel no desktop**, que é do laço do desktop.
- **Escrever no git.** Preparar, commitar, trocar de branch e descartar são da
  pessoa, no git dela, ou do agente pelo `bash`, sob a fronteira.
- **Diff contra a branch principal** (o que um PR mostraria). Precisa decidir
  contra qual base e com que `merge-base`; fica para quando alguém pedir.
- **Renomeação.** A lista roda com `--no-renames`: um arquivo renomeado aparece
  como um apagado e um novo, que é verdade e é simples de ler.
