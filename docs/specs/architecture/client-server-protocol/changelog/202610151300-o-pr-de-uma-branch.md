# O PR de uma branch, lido e aberto pelo `gh`

**Data:** 2026-10-15
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: duas rotas em
§4, a §5.5.1, dois códigos em §8, quatorze linhas novas em §9)
**Fonte:** pedido do dono, em 2026-10-15: o painel "Alterações" do desktop
ganha "Criar PR" e o estado do CI por sessão, como o painel do Claude Code
desktop — diff +N −M, "Criar PR", a lista de PRs com o ponto do CI. Regra do
dono: criar PR **sempre** pede confirmação explícita no app (título, corpo,
base e branch à vista, e a pessoa clica Confirmar); o app **nunca** faz merge,
nunca empurra com força, nunca liga auto-merge.

## O problema

O desktop mostra o que mudou numa pasta (`GET /v1/git`, 202610131900), e para
aí. Saber se a branch daquela conversa já tem PR, e se o CI dele está verde, é
abrir o navegador; abrir o PR é ir ao terminal, `git push -u`, `gh pr create`.
O registro de 202610131900 deixou "escrever no git" de fora de propósito:
preparar, commitar e trocar de branch continuam da pessoa. Abrir um PR é outra
coisa — é o passo em que a pessoa entrega o que já commitou —, e o dono pediu
que ele esteja no painel, sempre confirmado.

## O que mudou

- **`GET /v1/git/pr?workspace=`** responde a branch de agora, a base que um PR
  novo teria, e o PR da branch, se houver: número, endereço, título, estado
  (`open`, `closed`, `merged`), rascunho, base, head, e as checagens somadas
  numa palavra (`pass`, `fail`, `pending`, `none`) e em contagens.
- **`POST /v1/git/pr`** com `{workspace, title, body, base, draft}` empurra a
  branch de agora com `git push -u origin <branch>` e abre o PR com
  `gh pr create`. Responde `201` com o PR lido de volta.
- **Dois códigos de erro:** `gh_unavailable` (503) e `pr_refused` (409).
- **`vcs.ReadPR` e `vcs.CreatePR`**, ao lado do `vcs.ReadTree`; o cliente Go
  ganha `ReadGitPR` e `CreateGitPR`.

## Como, e por quê assim

- **No daemon, e não no processo principal do Electron.** Pelo mesmo motivo do
  registro de 202610131900: o daemon já é quem lê o git da pasta, pode estar
  noutra máquina que não a da janela, e uma segunda cópia das regras (qual é a
  branch, se HEAD está destacado) no cliente deixaria de seguir a primeira. E a
  TUI ganha a mesma superfície sem escrever nada de novo.
- **Pelo `gh`, não pela API do GitHub direto.** O `gh` é o que a pessoa já tem
  logado; o login, o host (GitHub Enterprise inclusive) e o token ficam com ele.
  O daemon não guarda, não lê e não transporta credencial nenhuma — uma rota
  que carregasse um token seria a primeira, e a lista de modelos já existe para
  provar que isso não acontece.
- **Uma chamada para ler, não duas.** O pedido falava em `gh pr view --json` e
  `gh pr checks`. O `statusCheckRollup` do `gh pr view` já traz cada checagem —
  check run com `status` e `conclusion`, status de commit com `state` —, então
  uma chamada basta, e as duas leituras nunca discordam sobre o mesmo instante.
- **A soma nunca é verde por cima de vermelho.** Falha vence pendente, pendente
  vence aprovada. `CANCELLED`, `TIMED_OUT`, `ACTION_REQUIRED`, `ERROR` são
  falha. `SKIPPED` e `NEUTRAL` contam à parte, em `skipped`, e sozinhos dão
  `none`, não `pass`: nada passou. Uma palavra que o GitHub inventar depois
  conta como falha — o erro de dizer vermelho a mais é visível e barato; o de
  dizer verde a mais é exatamente o que o ponto do CI existe para impedir.
- **A base padrão, sem rede quando dá.** Primeiro `origin/HEAD` do clone, que
  um `git clone` grava; só quando ele não existe o `gh repo view` pergunta ao
  GitHub. Ler o PR a cada foco da janela não pode custar duas idas à rede.
- **A ordem de abrir é fixa, e tudo o que recusa recusa antes do push:**
  pedido (título de uma linha até 256 caracteres, corpo até 65536 bytes, base
  que seja nome de branch), `gh` presente e com login (`gh auth status`), pasta
  em repositório, branch (não HEAD destacado), remoto `origin`, base, branch
  diferente da base, nenhum PR **aberto** para a branch, base conhecida no
  clone e ao menos um commit que ela não tem. Só então o push; só então o
  `gh pr create`. Um push que falhasse depois de um PR aberto, ou um PR aberto
  sobre uma branch que não foi, seriam meias ações que a pessoa não confirmou.
- **Nunca com força, e não há para onde cair.** Os argumentos do push são fixos
  (`push -u origin <branch>`); push recusado pelo remoto — branch divergente,
  proteção, credencial — é `pr_refused` com o que o git disse, e o remoto fica
  como estava. Nenhum `--force`, `--force-with-lease`, `+refspec`, `--auto`,
  `--fill` nem `merge` existe no código, e o teste confere a lista de
  argumentos que o `gh` recebeu.
- **Os ganchos de `pre-push` do repositório rodam.** Diferente do
  `core.fsmonitor` (202610131900), que rodaria só por alguém olhar o painel,
  aqui a pessoa pediu o push e confirmou: o push é o dela, com os ganchos que
  o push dela teria no terminal. Desligá-los seria o daemon empurrar o que o
  repositório disse para não empurrar.
- **Sem shell, sem terminal, sem herdar outro repositório.** `exec.Command`
  com os argumentos um a um; título, base e head vão como `--title=…`, para um
  título que comece com `-` não virar flag; o corpo vai pela entrada
  (`--body-file=-`), não pela linha de comando. O ambiente do `gh` perde
  `GH_REPO`, `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, pager e navegador, e
  ganha `GH_PROMPT_DISABLED=1`, `GIT_TERMINAL_PROMPT=0`, sem cor e sem aviso de
  atualização: o repositório é o da pasta, e ninguém espera uma pergunta que
  não tem quem responda. `GH_TOKEN`, `GH_HOST` e a pasta de configuração do
  `gh` ficam — são o login.
- **Prazos:** 20 s para ler (o `gh` vai à rede), 2 min para abrir (um push e
  duas idas ao GitHub). Prazo esgotado é dito como prazo, não como o que o
  comando cancelado escreveu.
- **Por workspace, como `/git`.** A conversa terminada é quando alguém quer
  abrir o PR do que ficou.
- **Falha dita, nunca "sem PR".** `gh` ausente e `gh` sem login são
  `gh_unavailable`, a mensagem diz qual e como resolver; o `gh` sai com 4
  quando pede login, e isso é lido. Branch sem PR é `pr` ausente, um estado
  comum e dito — nunca o que sobra de uma falha.
- **Confirmação é do cliente, e é contrato.** O daemon não sabe se uma pessoa
  viu o pedido; o protocolo diz que `POST /git/pr` é o que uma pessoa confirmou,
  e o cliente Go repete isso na documentação de `CreateGitPR`. O desktop mostra
  título, corpo, base e branch numa folha antes de mandar.

## As invariantes

As quatorze linhas novas em §9 são reivindicadas onde cada metade está — a
leitura e a abertura em `internal/vcs`, contra git e remoto de verdade e um
`gh` de mentira no `PATH` (`internal/vcs/testdata/fakegh`), e as rotas em
`internal/server`: `TestThePullRequestOfTheBranchIsReadWithItsChecks`,
`TestABranchWithNoPullRequestSaysSo`,
`TestTheDefaultBranchComesFromGitHubWhenTheCloneDoesNotSay`,
`TestOneFailingCheckMakesTheSummaryFail`,
`TestCreatingAPRPushesTheBranchThenOpensIt`, `TestADraftIsOpenedAsADraft`,
`TestOpeningAPRAnswersIt`, `TestARejectedPushIsSaidAndNeverForced`,
`TestAPRThatCannotBeOpenedIsRefusedBeforeAnyPush`,
`TestWithoutGhThePRIsNotReadOrOpened`, `TestThePRRoutesSayWhyWithACode`,
`TestAPRRequestIsValidatedBeforeAnythingRuns`,
`TestGhRunsWithoutAPromptOrAnInheritedRepository` e
`TestThePRRoutesRefuseAWorkspaceTheyCannotRead`.

Rotas, tipos e códigos novos, nenhum contrato existente mudado: MINOR.

## O que fica de fora

- **O painel no desktop**, que é do laço do desktop.
- **Merge, auto-merge, fechar, reabrir, pedir revisão.** Por regra do dono.
- **Commitar.** O PR leva o que já está commitado; o que está só na árvore
  continua da pessoa. O cliente diz quantos arquivos ficam de fora.
- **Outro remoto que não `origin`, e fork.** O push vai para `origin`; o
  `gh pr view` acha o PR pela branch. Fica para quando alguém pedir.
- **A TUI.** A superfície está pronta para ela; o comando é outra PR.
