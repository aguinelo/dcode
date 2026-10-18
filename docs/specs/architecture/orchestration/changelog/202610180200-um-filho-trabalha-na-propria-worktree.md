# Um filho trabalha na própria worktree

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §4, §10, §13, §14; `.config` §1, §2; `.i`,
passo 6).

## O que mudou

O passo 6 da primeira fatia. A orquestradora despacha filhos: `Orchestrator.Delegate` valida o lote
inteiro — `owns`, cruzamento entre itens e contra os `Leases` dos vivos, `max_children`, a lista de
perfis do papel e o teto de tokens — antes da primeira worktree, e recusa com o código da §5.4,
`dispatch.refused` no log e `session.error` na orquestradora. Cada filho ganha worktree e branch a
partir da ponta da main, sob `orchestrator.dir`, e uma sessão gravada `RoleChild`; o primeiro turno é
submetido e o `delegate` volta sem esperar. As ferramentas `delegate`, `children`, `steer_child` e
`close_child` ficam em `internal/tools/orchestrate.go`; a negação de git que grava, em
`internal/tools/gitwrite.go`. O harness lê os eventos de cada sessão do projeto (`Registry.Observe`):
os tokens de todo turno somam em `State.Spent`, a aprovação do filho o põe em `awaiting_person`, e o
fim de cada turno dele é commitado com os caminhos nomeados e dito em `child.completed`, com o que o
filho disse como `claim`. `GET /v1/sessions/{id}/children` e o `DELETE` de filho respondem; no boot,
os filhos não terminais reabrem sob o próprio id. Passam a ser lidas `max_children`,
`child_profiles`, `branch_prefix`, `token_ceiling` (na raiz do projeto) e `closed_retention` (onde o
daemon nasce).

## O que o código decide onde a `.p` deixava a forma em aberto

- **O id do filho nasce antes da sessão.** A worktree é `children/<id>/` e a branch termina no id;
  a sessão precisa da worktree para existir. Por isso `ChildRequest` ganha `ID`, cunhado pela
  orquestradora com `session.NewID`, e o daemon constrói a sessão sob ele (`buildAs`).
- **Os seis do id na branch são os últimos.** O `.config` dizia os seis primeiros; num id de sessão
  os primeiros são o relógio em milissegundos, e se repetem entre os filhos de um lote. Os últimos
  são a parte aleatória.
- **A `Spawner` ganha `SpawnChild`, `Submit` e `Steer`.** O `steer_child` é turno novo quando o
  filho está parado ou `failed`, e `steer` do turno corrente quando ele roda; os dois caminhos são do
  daemon. O steering vai ao log como `dispatch` com o texto, porque é mais trabalho despachado.
- **`done` deixa o filho em `working`, com o fim dito.** A transição para `in_review` é a da esteira
  (§7), que começa no passo 8; até lá o filho vivo continua contando em `max_children`.
- **`owns` que nomeia a árvore inteira (`.`) é recusado**, como o `OutsideOwns` do passo 8 já lia:
  dono de tudo é dono de nada em particular. A conferência de `owns` de quem escreve pelas
  ferramentas usa o mesmo `OutsideOwns`, uma regra só; a do `bash` é a do `git status` do passo 8.
- **O commit do harness usa o `Uncommitted` do passo 8** para saber o que mudou, e o `title` como
  mensagem; o `pr_title` e o `pr_body` vão ao log no `dispatch`, para o passo 10.
- **`Spent` não é decisão.** Os tokens de um turno da orquestradora reescrevem o `state.json` sem
  linha no log; o do filho entra na linha de `child.completed`. O teto pergunta uma vez por janela:
  o `ceiling.reached` e a `Question{kind: ceiling}` saem na primeira recusa, e as seguintes só dizem
  `dispatch.refused`. A resposta é `Orchestrator.Answer`; a rota de `/questions` é do passo 10.
- **Perfil da lista que o `models.toml` não tem** recusa o filho na criação (`model_not_allowed`),
  e ele vai a `failed` dizendo isso: só o daemon conhece o `models.toml` da worktree.

## Invariantes e o que fica

Sobem para a §13 onze linhas da §14, cada uma com o teste que a reivindica em
`internal/orchestrate/invariants_test.go`. Ficam: o diff fora do `owns` sem commit e sem revisão
(passo 8, que agora tem o commit para não fazer), toda transição com linha antes do evento, todo fim
de turno com um nome da RN-7, e o relato no começo de um turno da orquestradora (passo 7).
