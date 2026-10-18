# Planning: Orquestrador — primeira fatia, o núcleo observável

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202610132000-orchestration.r.spec.md` (aprovado em
> 2026-10-17, #514); a ordem das fatias é a §10 dele, as decisões as §§12–14.
> Configuração em `202610132000-orchestration.config.spec.md`; passos e testes
> obrigatórios em `202610132000-orchestration.i.spec.md`.

## 1. Nível de estabilidade

**`experimental`**. A superfície pública é do protocolo (`202608072240-client-server-protocol`,
abaixo de 1.0): aditiva, campos novos `omitempty`, nenhum muda de sentido — **MINOR**. `stable`
pede a segunda fatia e um projeto real de ponta a ponta com o log lido (RN-11).

## 2. O que esta fatia é, e o que fica para a segunda

**Pré-requisito, passo 0:** o changelog em fragmentos (Q-A), já na main (#515). Esta spec não o
descreve; o filho escreve o fragmento dele em `changelog.d/`, dentro do próprio `owns`.

**Dentro:** uma orquestradora por projeto, viva no daemon e em disco; filhos como sessões com
`parent_id`, worktree e branch próprias, `owns` disjuntos; `child.*` no fio; o log de decisões em
JSONL e `dcode decisions`; revisor só leitura; o gate rodado pela orquestradora no commit exato;
lugar de integração próprio; merge **só em passo a passo** — PR por filho, squash pelo `gh` com
`--match-head-commit`, merge local sem forge; o teto de tokens, que para o despacho; a lista viva
(painel do desktop, texto na TUI); os cenários com modelo roteirizado. Decisões do dono de
2026-10-17 sobre este plano: no changelog da família, `202610172200`.

**Fora — segunda fatia (§10 do `.r`), com ponteiro na `.i`:** `launchd` (RN-2); supervisionado e
autônomo (RN-3); a lista "sempre perguntar" como negação de `internal/policy` (RN-4); tetos de custo
e de tempo (RN-3); custo em dinheiro e a tabela de preços (RN-9); deploy (RN-13); notificações
(RN-10); contratos com modelo real (§12). Em passo a passo todo merge pergunta, e nada desta fatia
faz deploy, tag, release ou force-push: o que a RN-4 lista já passa pela pessoa.

## 3. Onde mora o código

```
internal/orchestrate/  project orchestrator lease store decisions worktree pipeline gate forge (.go)
internal/tools/orchestrate.go  internal/protocol/orchestrate.go  internal/server/orchestrate.go
internal/session/ (Reopen, SubmitOrigin, aprovação sem prazo)  internal/vcs/ (worktree, merge, squash)
pkg/client/orchestrate.go  cmd/dcode/decisions.go  internal/tui/agents.go
desktop/src/renderer/AgentsPanel.tsx
```

`internal/orchestrate` não importa `internal/server` nem `internal/app`: fala com o daemon por uma
`Spawner`, como o `Runner` do `/loop` fala por uma `Transport`
(`internal/loop/loopcommand/runner.go`), e por isso é testado sem modelo e sem socket. O
`app.Daemon` implementa a `Spawner`.

## 4. Tipos do pacote

```go
package orchestrate
type Registry struct{ /* dir, spawner, forges, mu, projects map[string]*Orchestrator */ }
func (r *Registry) Open(workspace string) (*Orchestrator, bool, error) // idempotente; bool: criado
func (r *Registry) Reload(ctx context.Context) []error           // no boot, §8
// Orchestrator conducts one project. No method waits for a child.
type Orchestrator struct{ /* state State, store *Store, log *DecisionLog, leases Leases */ }
type Spawner interface {
	SpawnOrchestrator(p protocol.Project) (protocol.Session, error)
	SpawnChild(req ChildRequest) (protocol.Session, error)
	SpawnReviewer(req ReviewRequest) (protocol.Session, error)
	Reopen(sessionID, reason string) (protocol.Session, error)
	Submit(sessionID, text, origin string) error
	Close(sessionID string) error
}
type ChildRequest struct{ ParentID, Worktree, Profile, Task string; Owns []string }
type ReviewRequest struct{ ParentID, ChildID, Worktree, Profile, Base, Commit, Task, Diff string; Owns, Files, Specs, MissingSpecs []string; DiffBytes int; DiffCut bool }
type State struct { // state.json, um por projeto
	V              int                 `json:"v"` // 1
	Project        protocol.Project    `json:"project"`
	Children       []protocol.Child    `json:"children"`
	Questions      []protocol.Question `json:"questions"`
	Queue          []string            `json:"queue"` // ids de filho, serial
	Main           protocol.MainHealth `json:"main"`
	LogSeq         uint64              `json:"log_seq"`
	Spent          int64               `json:"spent"` // tokens na janela do teto, §10
	PendingReports []string            `json:"pending_reports,omitempty"`
}
type Store struct{ /* path */ }
func (s *Store) Load() (State, error)
func (s *Store) Save(st State) error // temp + fsync + rename
type Decision struct{ /* §9.1 */ }
type DecisionLog struct{ /* f, seq */ }
func (l *DecisionLog) Record(d Decision) (uint64, error)
type Leases struct{ /* id do filho → owns */ }
func OwnsOverlap(a, b string) bool
type Gate struct{ Command string; Timeout time.Duration }
func (g Gate) Measure(ctx context.Context, dir, commit string) (protocol.GateResult, error)
type Forge interface {
	Name() string // "github" | "local"
	OpenPR(ctx context.Context, branch, title, body string) (protocol.ChildPR, error)
	Merge(ctx context.Context, c protocol.Child, measured string) (string, error)
	MainHealth(ctx context.Context, commit string) (protocol.MainHealth, error)
}
```

`OwnsOverlap` é conservadora: cada entrada vira o prefixo literal até o primeiro metacaractere de
`policy.Glob`, cortado num separador; duas se cruzam quando um prefixo contém o outro. Recusa
`a/*.go` contra `a/*.md` — o preço de nunca deixar passar um cruzamento real, dito na mensagem.

## 5. Protocolo

### 5.1 Rotas

| Método e caminho | Corpo | Resposta | Erros |
|---|---|---|---|
| `POST /v1/projects` | `OpenProjectRequest{workspace}` | `Project`; 201 criado, 200 existente | `workspace_invalid`, `orchestrator_unavailable` |
| `GET /v1/projects` | — | `ListProjectsResponse{projects}` | — |
| `GET /v1/projects/{id}` | — | `Project` | `project_not_found` |
| `GET /v1/projects/{id}/orchestrator` | — | a `Session` da orquestradora | `project_not_found` |
| `GET /v1/sessions/{id}/children` | — | `ListChildrenResponse{children}`: vivos, e fechados dentro da retenção | `session_not_found` |
| `GET /v1/sessions/{id}/questions` | — | `ListQuestionsResponse{questions}` | `session_not_found` |
| `POST /v1/sessions/{id}/questions/{qid}` | `AnswerQuestionRequest{answer, note}` | a `Question` respondida | `question_not_found`, `question_already_answered`, `invalid_input` |

`DELETE /v1/sessions/{id}` num filho é o fechar da pessoa: `closed`, worktree removida, `owns`
liberado, linha no log. Na orquestradora é `invalid_input` nesta fatia; arquivá-la é da segunda.
`/children` de uma sessão sem filhos é lista vazia; o filho `explore` não é sessão e não aparece.
Quem cria é o `POST`, não o `GET` que o `.r` §5 descrevia; o porquê está no changelog da família.

### 5.2 Tipos no fio

Tag JSON é o campo em snake_case; `omitempty`: os de `Session`, `Conversation`, `TurnStarted`, e os `?`.

```go
// Session e Conversation ganham:
ParentID, ProjectID, Role string; Owns []string // Branch e Model já existem
const RoleOrchestrator, RoleChild, RoleReviewer = "orchestrator", "child", "reviewer"
// TurnStarted ganha Origin: "" é a pessoa.
const OriginChildren = "children"
// Fim de turno novo, ao lado dos Stop* de hoje.
const StopDaemonRestart = "daemon_restart"

type Project struct {
	ID             string    // 24 hex de sha256 da raiz
	Root           string    // raiz da worktree principal, links resolvidos
	Main, Forge    string    // a branch main; "github" | "local"
	Autonomy       string    // "step", sempre, nesta fatia
	OrchestratorID string
	Created        time.Time
}
type Child struct {
	ID, ParentID, Title, Profile, Model, State string // ID = id da sessão; State da §6
	Reason?, Waiting? string // fim do último turno (RN-7); "approval" | "question" | "merge"
	Owns             []string
	Branch, Worktree string
	Base             string // commit da main de onde a branch saiu
	Head?            string // último commit que o harness fez
	Returned?        int; Return? *Return
	Review?          *Verdict; Gate? *GateResult; PR? *ChildPR
	Usage            Usage // somado sobre os turnos
	Started, Ended?  time.Time
}
type Return struct{ Why, Phase? string; Paths? []string } // Why: outside_owns | review | conflict | gate_red | person; Phase: red | fix
type Verdict struct{ SessionID, Commit, Verdict string; Findings? []Finding } // approve | changes_requested | missing
type Finding struct{ Path string; Line? int; What, Why string }
type GateResult struct {
	Commit, Tree, Base, Command string // o commit medido, a árvore dele, a main sob ele
	ExitCode, DurationMS        int
	Result                      string // "green" | "red" | "timeout" | "undeclared"
	OutputTail?, OutputPath?    string // os últimos 4 KiB; a saída inteira em disco
	At                          time.Time
}
type ChildPR struct{ Number?, URL?, CI? } // CI na gramática de vcs.Checks; Number 0 no local
type MainHealth struct{ Commit, Health string; Checked time.Time } // Health: green | red | pending | unknown
type Question struct {
	ID, Kind, Text, Recommended, Why string // Kind "merge" | "design" | "ceiling"; Recommended é um Option.ID
	ChildID?                          string
	Options                           []QuestionOption
	Sensitive?                        []string // §7.5
	Asked                             time.Time
	Answer?, Note?                    string; Answered? time.Time
}
type QuestionOption struct{ ID, Label string }
```

### 5.3 Eventos — todos no log da orquestradora

O progresso do filho fica no log dele; aqui entra só o que a lista viva e a conversa precisam.

| Tipo | Payload | Quando |
|---|---|---|
| `child.started` | `ChildChanged{child}` | a sessão e a worktree do filho existem |
| `child.state` | `ChildChanged{child}` | transição da §6, ou PR, CI ou uso que mudou |
| `child.completed` | `ChildCompleted{child_id, turn_id, reason, conclusion, truncated, wrote, unwritten, usage}` | todo turno de filho que termina; `conclusion` cortada em `DelegateMaxResultBytes` |
| `review.verdict` | `Verdict` e `child_id` | o revisor chamou `verdict`, ou terminou sem (`missing`) |
| `gate.measured` | `GateResult` e `child_id` | toda medição, verde ou não |
| `question.asked` | `Question` | pergunta entrou na fila |
| `question.answered` | `Question` | a pessoa respondeu |
| `merge.completed` | `MergeCompleted{child_id, method, pr, merged, measured}` | `merged` é o commit na main; `measured`, o do gate |
| `merge.refused` | `MergeRefused{child_id, code, reason}` | `main_red`, `gate_stale`, `gate_undeclared`, recusa do forge |
| `main.health` | `MainHealth` | depois de cada merge, até o CI terminar |

### 5.4 Códigos de erro novos

| Código | HTTP | Onde | Quando |
|---|---|---|---|
| `project_not_found` | 404 | rota | id que o daemon não tem |
| `orchestrator_unavailable` | 503 | rota | `orchestrator.enabled` falso, ou gravação de sessões desligada; a mensagem diz qual |
| `owns_overlap` | 409 | `delegate` | `owns` que se cruzam, no lote ou com filho vivo; os caminhos na mensagem |
| `outside_owns` | 409 | esteira | diff do filho fora do `owns`; os caminhos na mensagem |
| `max_children_reached` | 503 | `delegate` | o lote passaria de `orchestrator.max_children` vivos |
| `model_not_allowed` | 400 | `delegate` | perfil fora da lista do papel |
| `main_red` | 409 | esteira | merge com a main vermelha ou desconhecida |
| `gate_stale` | 409 | esteira | main ou branch andou entre o gate e o merge |
| `gate_undeclared` | 409 | esteira | nem `orchestrator.gate_command` nem `verify.command` |
| `ceiling_reached` | 409 | `delegate` | `State.Spent` chegou a `orchestrator.token_ceiling` (§10) |
| `child_in_pipeline` | 409 | `steer_child` | filho em `in_review` ou `pr`: a esteira é dona dele |
| `question_not_found` | 404 | rota | id que a fila não tem |
| `question_already_answered` | 409 | rota | o primeiro que responde leva |

Recusa de ferramenta volta ao modelo como resultado `ok: false` com o código no texto; o mesmo fato
vai à pessoa como `session.error` na orquestradora.

## 6. A máquina de estados de um filho (RN-9)

`dispatched` (despachado), `working` (trabalhando), `awaiting_person` (esperando você), `in_review`
(em revisão), `pr` (PR), `merged` e `closed` (fechado), terminais, e `failed` (falhou), que espera.

| De | Para | Quando |
|---|---|---|
| `dispatched` | `working` | worktree e sessão criadas, primeiro turno submetido |
| `dispatched` | `failed` | worktree ou sessão não pôde ser criada; `reason: error`, o motivo dito |
| `working` | `awaiting_person` | o filho pediu aprovação; `waiting: approval` |
| `awaiting_person` | `working` | a pessoa respondeu |
| `working` | `in_review` | turno terminou `done`; a esteira começa (§7) |
| `working` | `failed` | turno terminou `incomplete`, `max_iterations`, `error`, `interrupted` ou `daemon_restart` |
| `in_review` | `working` | devolvido (`Return`): `outside_owns`, revisão, conflito, gate vermelho |
| `in_review` | `pr` | revisão `approve`, gate `green`, PR aberto; pergunta de merge na fila |
| `pr` | `in_review` | `gate_stale`: mede de novo |
| `pr` | `working` | a pessoa respondeu `return` |
| `pr` | `merged` | merge feito e registrado |
| `failed` | `working` | `steer_child` |
| não terminal | `closed` | `close_child`, ou `DELETE` da pessoa; `reason: cancelled` |

Nada mais transita. `in_review` cobre revisão, atualização e gate (a RN-9 tem oito estados); o
passo corrente se lê por `Review` e `Gate` preenchidos. Além de `orchestrator.max_returns`
devoluções: `failed`, `reason: incomplete`, "devolvido N vezes", relato à orquestradora.

## 7. A esteira — o que decide se algo entra na main

Roda **no harness**: ao fim `done` de um turno do filho, os passos correm em ordem e nenhum é
pulado; o modelo lê os resultados e decide. **Emenda o `.r` §6**, que os listava como ferramentas
(decisão 2 do dono; o porquê no changelog `202610172200`). A orquestradora os vê por `children` e pelos relatos.

**7.1 Commit e `owns`.** O filho não faz commit; o harness faz, ao fim de cada turno. Lê `git status --porcelain` na worktree **antes** do commit: caminho fora do `owns` (`policy.Glob`) devolve **sem commit e sem revisão** (`Return{outside_owns, paths}`). Dentro, adiciona **os caminhos nomeados** — nunca `-A` — e commita com o `title` como mensagem; `Head` é esse commit. Log: `owns.checked`, com os caminhos.

**7.2 Revisão.** Uma sessão `RoleReviewer`, filha da orquestradora, na worktree do filho, com `read`, `glob`, `grep`, `symbol` e `verdict` (`ReviewerRegistry`) — sem escrita, sem `bash`. `AssembleReview` monta a tarefa, o `owns`, as specs que a tarefa nomeia e o diff `Base..Head` até `orchestrator.review_max_diff_bytes`; acima disso, o diff sai inteiro (`DiffCut`), dito, com a lista de arquivos e a instrução de lê-los; arquivo fora do `owns` no diff é `outside_owns` sem revisor. `verdict{verdict, findings, why}`. Turno sem `verdict` é `missing` (`ErrNoVerdict`) e roda de novo uma vez; o segundo `missing` deixa o filho em `in_review`, com relato à orquestradora.

**7.3 Devolução com teste vermelho primeiro (RN-5).** `changes_requested` volta **ao mesmo filho** em duas fases. **`red`:** o turno pede só o teste que mostra o defeito e termina com `red_test{command}`; o harness commita, roda o comando nesse commit numa worktree descartável e exige saída diferente de zero — `red_test.confirmed`, ou, verde, de novo `red`, dizendo; turno `red` sem `red_test` também volta a `red`. **`fix`:** o conserto; ao fim, a esteira recomeça em 7.1. Gate vermelho devolve direto em `fix`: a saída dele já é a medição vermelha. A passagem de `red` a `fix` é metade de uma devolução e não conta em `max_returns`.

**7.4 Atualização e gate, no lugar de integração** — uma worktree do daemon por projeto, destacada, nunca a da pessoa:

1. `git fetch` (forge `github`); `M` é a ponta da main.
2. Checkout de `Head` e `git merge --no-ff M` — **merge da main, nunca rebase**. Conflito:
   `git merge --abort`; o harness repete o merge **na worktree do filho** deixando os marcadores e
   devolve (`Return{conflict, paths}`); ao fim do turno seguinte o harness conclui o merge com o
   commit dele. A orquestradora não resolve conteúdo (Q-H).
3. `H` é o commit resultante. `Gate.Measure(H)` com `orchestrator.gate_command`, senão `verify.command`, senão
   `gate_undeclared` e nada segue: numa worktree descartável em `H`, sem `GIT_*`, até `orchestrator.gate_timeout`.
4. `gate.measured` traz `commit H`, `tree`, `base M`, comando, saída e duração; o que o filho disse do próprio teste
   (`child.completed.conclusion`) vai como `claim` — alegação ao lado do medido, nunca no lugar dele.

**7.5 PR e a pergunta de merge.** Gate `green`: o harness empurra a branch
(`git push -u origin <branch>`, nunca `--force`) e abre o PR com `pr_title` e `pr_body` mais um
bloco fixo de evidência (commit medido, comando, resultado, veredito). Forge `local`: sem push, sem
PR. Então `Question{kind: merge, options: merge | return | close, recommended: merge}`, com
`Sensitive` preenchido quando o diff toca `orchestrator.sensitive` ou passa de
`orchestrator.sensitive_diff_lines`. Só a pessoa responde: a orquestradora não tem ferramenta para
isso.

**7.6 Merge, fila serial.** A resposta `merge` põe o filho na `Queue`; um por vez, na ordem:

1. `State.Main` vermelha ou desconhecida → `merge.refused{main_red}`, o filho fica em `pr`; salvo
   item de lote com `fixes_main: true`.
2. `fetch` de novo. Ponta da main ≠ `GateResult.Base`, ou da branch ≠ `GateResult.Commit` →
   `gate_stale`, de volta a `in_review` e a 7.4.
3. **GitHub:** `gh pr merge <n> --squash --match-head-commit <H>`; a branch remota, que é do
   harness, é apagada depois. **Local:** recusado com a main em checkout em qualquer worktree
   (`git worktree list --porcelain`), dizendo onde; senão `git merge --squash H` na integração,
   commit `S`, e `git update-ref refs/heads/<main> S M` — a troca condicional é a trava do commit
   medido, sem servidor. Na primeira vez que um projeto cai em `local`, o log (`forge.local`) e a
   conversa dizem o que se perde: o PR para ler, o CI do forge e a trava no servidor.
4. `merge.completed`; worktree do filho removida, `owns` liberado, sessão fechada, estado `merged`.
5. `main.health` até o CI do commit merged terminar ou `orchestrator.ci_timeout`. `red` **para a
   fila**; `pending` no tempo-limite vira `unknown` e também para. Só `green` libera o próximo.

Nenhuma rota, ferramenta ou função de `internal/vcs` desta fatia faz rebase, `push --force` ou
reescreve um commit que existe.

## 8. Persistência e reinício

Em `<orchestrator.dir>/<project-id>/`: `state.json` (o `State`, reescrito atômico a cada transição), `decisions.jsonl`
(§9, só acréscimo), `gates/<commit>.log`, `integration/` (o lugar de integração) e `children/<child-id>/` (a worktree de cada filho).

**A ordem de cada transição:** linha no log → `state.json` → evento no fio. Linha que não grava
impede a transição, e a falha vai à conversa como `session.error`. O estado em disco nunca está à
frente do log.

As conversas — orquestradora, filhos, revisores — são sessões gravadas pelo `Record` de hoje; sem
`record.enabled`, `orchestrator_unavailable`.

**No boot**, `Registry.Reload` lê cada `state.json` e, por projeto:

1. reabre a orquestradora **sob o mesmo id** (`Spawner.Reopen`): histórico reconstruído do registro
   como o `Resume` faz, `Seq` continuando do último;
2. reabre cada filho não terminal sob o mesmo id. Turno aberto no registro ganha
   `turn.completed{reason: daemon_restart}` no log dele; o filho vai a `failed` com esse motivo, na
   lista e no log, e um relato entra na caixa da orquestradora, que decide continuar ou redespachar;
3. a esteira retoma do passo persistido: revisão ou gate em curso **rodam de novo** (gate só conta
   terminado); merge em curso é conciliado pelo forge (`gh pr view` diz `MERGED` → `merged`; senão
   `pr`); perguntas pendentes continuam pendentes;
4. `daemon.restart` no log, com o que reabriu, o que não pôde e a linha que `Torn` cortou.

`state.json` ilegível: o projeto não abre, o erro vai ao log do daemon e `Reload` o devolve — nunca um projeto vazio.

## 9. O log de decisões

### 9.1 Esquema — uma linha por decisão

| Campo | Tipo | Obrigatório | Sentido |
|---|---|---|---|
| `v` | int | sim | versão do formato: 1 |
| `seq` | uint64 | sim | por projeto, de 1, sem buraco nem repetição, através de reinícios |
| `at` | RFC 3339, UTC | sim | — |
| `project` | string | sim | `Project.ID` |
| `kind` | string | sim | da lista abaixo |
| `child`, `session` | string | quando há | o filho; a sessão que agiu |
| `why` | string | sim, não vazio | do modelo quando ele escolheu (`why` das ferramentas), do harness quando a regra escolheu |
| `claim` | string | não | o que um filho **disse** — nunca lido como prova |
| `measured` | objeto | nas medições | `GateResult`, `Verdict`, `MainHealth` |
| `data` | objeto | não | o resto do fato, por `kind` |

`kind`: `project.opened`, `dispatch`, `dispatch.refused`, `model.chosen`, `worktree.created`,
`hooks.changed`, `child.state`, `child.completed`, `child.returned`, `owns.checked`,
`review.verdict`, `red_test.confirmed`, `branch.updated`, `gate.measured`, `pr.opened`,
`question.asked`, `question.answered`, `merge.queued`, `merge.completed`, `merge.refused`,
`main.health`, `ceiling.reached`, `forge.local`, `worktree.removed`, `child.closed`,
`daemon.restart`.

Nenhuma linha passa de 64 KiB: o maior entre `why`, `claim`, `measured` e `data` é cortado com
`…[N bytes cortados]` (`measured` ou `data` cortados viram string JSON); o que nem assim cabe é erro.
`hooks.changed` sai quando `core.hooksPath`, lido depois de criar uma worktree, difere do lido antes,
com os dois valores; a orquestradora não o reescreve. **Durabilidade:** `Record` só devolve o `seq`
depois do `fsync`; escrita que falha é erro, é desfeita e não gasta `seq`. Ao reabrir, a linha final
sem fim de linha (queda no meio do `Record`) é cortada e devolvida por `Torn`; linha completa
ilegível ou `seq` fora de ordem impede o log de abrir, com o número da linha.

### 9.2 `dcode decisions`

```
dcode decisions [--project <caminho>] [--child <id>] [--kind <k>[,<k>]] [--since <dur>] [--follow] [--json]
```

Lê o arquivo direto, **sem daemon**: o log tem de ser legível quando o que falhou foi o daemon.
`--project` padrão é o diretório corrente, resolvido como o `POST /v1/projects` resolve. Texto, uma
linha por decisão: `21:04:05  #42  gate.measured  índice  red  make check (exit 2)  why: …`;
`--json` devolve as linhas como estão. Linha ilegível é dita em stderr com o número, o resto é
impresso, e a saída é 1; `seq` com buraco e linha final rasgada também. Nunca escreve no log.

## 10. Ferramentas e registros

**A orquestradora** é uma sessão em modo `plan` na raiz do projeto: lê, conversa e conduz; **não
escreve na árvore**. Quem escreve é filho, numa worktree. Tem as ferramentas de leitura de hoje,
`explore` (Q2 da #461) e estas, nenhuma esperando filho:

| Ferramenta | Entrada | Volta |
|---|---|---|
| `delegate` | `{items: [{title, task, owns, model, pr_title, pr_body, fixes_main, why}]}` | os `Child` criados, já `dispatched` |
| `children` | `{}` | os `Child`, em texto determinístico |
| `steer_child` | `{child_id, text, why}` | ok: turno novo, ou `steer` do corrente |
| `close_child` | `{child_id, why}` | ok |
| `ask` | `{text, options, recommended, why, child_id}` | o id da pergunta |

`delegate` é **tudo ou nada**: valida o lote inteiro — `owns` não vazio e dentro da raiz,
`OwnsOverlap` entre itens e contra `Leases`, `max_children`, perfil na lista do papel, teto de
tokens — antes de criar qualquer worktree. `model` vazio é o primeiro de
`orchestrator.child_profiles`.

**O teto de tokens** (RN-3, trazido para esta fatia pelo dono). `State.Spent` soma `input`, `output`
e `cache_read` de toda sessão do projeto — orquestradora, filhos, revisores. Ao chegar a
`orchestrator.token_ceiling`, `delegate` é recusado com `ceiling_reached`, o fato vai ao log
(`ceiling.reached`) e à conversa, e entra `Question{kind: ceiling, options: continue | stop}`;
`continue` zera `Spent` e abre outra janela. Filho vivo termina o que faz: parar filho por teto, com
`budget`, é da segunda fatia, com os tetos de custo e de tempo.

**O filho** tem o registro de uma sessão comum **menos** `explore`, `delegate`, `children`,
`steer_child`, `close_child`, `ask` e `verdict`; `red_test` só na fase `red`. Tem `bash` na
worktree, pelo sandbox, gravável só nela. Uma regra de negação — não pergunta, diz o porquê — recusa
`git commit`, `merge`, `rebase`, `push`, `reset`, `checkout`, `switch`, `stash`, `config`,
`worktree`, `tag` e `branch`: git que grava é do harness. Aprovação de filho é anunciada com
`ExpiresAt` zero e **não expira**; nenhuma ferramenta da orquestradora a responde (ADR-02).

**O relato** (RN-1): `child.completed`, `merge.completed`, `merge.refused`, `main.health` vermelha e
`question.answered` entram na caixa (`State.PendingReports`). Com a orquestradora `idle`, a caixa
vira um turno `Origin: children` depois de `orchestrator.report_debounce`, com um texto montado pelo
harness, em inglês, acrescentado ao fim do histórico — nunca no meio de um turno, nunca reescrevendo
o prefixo (ADR-03). Com um turno em curso, espera ele terminar.

## 11. A lista viva

Só do fio: `GET /v1/sessions/{orq}/children` ao abrir, depois `child.*`, `question.*` e
`main.health` do SSE da orquestradora — nada de texto (D8). Fechado sai
`orchestrator.closed_retention` depois de `Ended`, pela resposta do daemon, não por relógio do
cliente; `failed` não sai sozinho. A forma no desktop (`AgentsPanel`, gramática de D41, PR e CI de
D60 e D61) e na TUI (`dcode orchestrate`; `dcode` continua a conversa comum) está nos passos 13 e 14
da `.i`.

## 12. Contratos comportamentais

Os sete do `.r` §9 são mediados e entram na segunda fatia, com fixture, juiz e limiar. A metade
determinística de `orchestrator-distrusts-the-claim` — nenhum merge sem gate verde — já é desta.

## 13. Invariantes verificáveis

- `seq` do log é contínuo por projeto, através de reinícios.
- Decisão sem `why`, ou de `kind` fora da lista da §9.1, é recusada e nada é escrito.
- Escrita do log que falha é erro devolvido e não gasta `seq`; nunca uma linha perdida em silêncio.
- Última linha sem fim de linha é cortada ao reabrir e devolvida por `Torn`; linha completa ilegível ou `seq` fora de ordem impede o log de abrir.
- Nenhuma linha do log passa de 64 KiB; o campo maior é cortado e diz quantos bytes perdeu.
- `dcode decisions` lê o log sem daemon e nunca o escreve; linha ilegível é dita com o número, e o resto é impresso.
- Um projeto tem **uma** orquestradora: dois `POST /v1/projects` da mesma árvore, por caminhos diferentes, devolvem o mesmo `Project`.
- A orquestradora volta do reinício do daemon **sob o mesmo id**, com o histórico inteiro.
- O gate roda numa worktree descartável no commit exato, sem `GIT_*` no ambiente, nunca na árvore da pessoa nem na do filho.
- A worktree descartável do gate sai sempre, até depois de um pânico.
- O que o filho disse do gate vai ao log como `claim`, ao lado do medido, e nunca muda o `result`.
- Sem comando de gate declarado nada roda, o resultado é `undeclared`, e `gate_undeclared` é dito.
- Medição de outro commit, ou sobre outra main, que não os do merge é `gate_stale`.
- O revisor não tem ferramenta que escreve.
- Revisão que termina sem `verdict` é `missing`, vai ao log, e nunca conta como aprovação.
- Diff acima de `review_max_diff_bytes` sai inteiro e é dito; nunca chega cortado ao revisor como se fosse o todo.
- Criar filhos volta com os ids antes de qualquer turno de filho terminar; com filhos rodando, a orquestradora fica `idle`, nunca `blocked`.
- Lote com `owns` que se cruzam, entre si ou com filho vivo, é recusado **inteiro**, com os caminhos em disputa, e nenhuma worktree é criada.
- Lote que passaria de `max_children` é recusado inteiro com `max_children_reached`.
- Perfil fora da lista do papel é recusado com `model_not_allowed`.
- O registro do filho não tem `explore`, `delegate`, `children`, `steer_child`, `close_child` nem `ask`.
- Comando git que grava, no `bash` do filho, é negado sem perguntar e com o porquê.
- Aprovação de filho não expira, e nenhuma ferramenta da orquestradora a responde.
- Criar a worktree não muda `core.hooksPath`; se mudou, `hooks.changed` diz os dois valores.
- Turno de filho aberto no reinício termina com `daemon_restart`, dito na lista e no log, e nunca some.
- Com `Spent` no teto de tokens, `delegate` é recusado com `ceiling_reached` e o porquê é dito.
- Fechado sai da lista depois da retenção; `failed` não sai sozinho.
- Diff do filho com caminho fora do `owns` volta ao filho sem commit e sem revisão.
- Defeito do revisor volta ao mesmo filho em fase `red`, e o conserto só é aceito depois de `red_test.confirmed`.
- Teste da fase `red` que passa não é aceito como vermelho, e conserto sem a fase `red` confirmada nunca chega ao gate.
- A segunda revisão seguida sem `verdict` deixa o filho em `in_review`, dito à orquestradora; não há terceira.
- Além de `max_returns` devoluções o filho vai a `failed`, `incomplete`, e a orquestradora é avisada.
- O gate da esteira mede o `Head` exato do filho, com `base` a ponta da main sob ele, e o que o filho disse ao lado como `claim`.
- Relato de filho entra no histórico da orquestradora só no começo de um turno dela, com `Origin: children`.
- Relatos dentro de `report_debounce` viram **um** turno, e o contexto antes dele é byte a byte o de antes (ADR-03).

## 14. Invariantes previstas — sem teste ainda; cada uma sobe para a §13 com o seu

- Todo fim de turno de filho tem um dos nomes da RN-7.
- O gate roda no commit `H` que inclui a main de agora, e `gate.measured` traz `commit`, `tree` e `base`.
- Sem gate verde não há PR nem merge, e main ou branch que andou entre o gate e o merge mede de novo.
- O merge no GitHub passa `--match-head-commit` com o commit medido.
- Merge local é troca condicional da ref da main, e é recusado com a main em checkout em qualquer worktree.
- Um merge por vez; com a main vermelha ou desconhecida, nenhum outro entra, salvo `fixes_main`.
- Conflito ao atualizar a branch volta ao filho dono dela; a orquestradora não resolve conteúdo.
- Nenhum caminho do código desta fatia roda `rebase`, `push --force` ou `commit --amend`.
- A integração nunca acontece numa worktree que a pessoa usa.
- Toda transição da §6 tem linha no log antes do evento; linha que não grava impede a transição.

## 15. Ordem de entrega

Cada linha é um PR, e o título não tem "and". Detalhe e teste obrigatório de cada passo na `.i`, e lá também as diferenças desta
ordem para a do `.r` §10, com o porquê.

| # | PR | Depende |
|---|---|---|
| 0 | `docs: the changelog is assembled from fragments` — #515, feito | — |
| 1 | `docs(orchestration): plan the first slice` — este | — |
| 2 | `feat(protocol): a session can have a parent` | 1 |
| 3 | `feat(session): a recorded session reopens under its own id` | 2 |
| 4 | `feat(orchestrate): every decision is logged with its reason` | 2 |
| 5 | `feat(orchestrate): a project has one orchestrator that survives the daemon` | 3, 4 |
| 6 | `feat(orchestrate): a child works in its own worktree` | 5 |
| 7 | `feat(orchestrate): a child's report starts the orchestrator's turn` | 6 |
| 8 | `feat(orchestrate): a reviewer reads every child's diff` | 6 |
| 9 | `feat(orchestrate): the orchestrator runs the gate itself` | 8 |
| 10 | `feat(orchestrate): a merge waits for the person` | 9 |
| 11 | `feat(cli): dcode decisions reads the log` | 4 |
| 12 | `test(orchestrate): scripted scenarios` | 10 |
| 13 | `feat(tui): the orchestrator lists its agents` | 7 |
| 14 | `feat(desktop): the live agent list` | 7 |

## 16. Changelog
- [202610172200 — A primeira fatia, planejada](changelog/202610172200-a-primeira-fatia-planejada.md)
- [202610172300 — A sessão gravada reabre sob o próprio id](changelog/202610172300-a-sessao-gravada-reabre-sob-o-proprio-id.md)
- [202610172330 — Toda decisão vai ao log com o porquê](changelog/202610172330-toda-decisao-vai-ao-log.md)
- [202610172350 — O log de decisões se lê sem o daemon](changelog/202610172350-o-log-de-decisoes-se-le-sem-o-daemon.md)
- [202610172358 — A orquestradora roda o gate ela mesma](changelog/202610172358-a-orquestradora-roda-o-gate.md)
- [202610180000 — Um projeto tem uma orquestradora, que sobrevive ao daemon](changelog/202610180000-uma-orquestradora-por-projeto.md)
- [202610180100 — Um revisor lê o diff de cada filho](changelog/202610180100-um-revisor-le-o-diff-de-cada-filho.md)
- [202610180200 — Um filho trabalha na própria worktree](changelog/202610180200-um-filho-trabalha-na-propria-worktree.md)
- [202610180300 — O relato de um filho começa o turno da orquestradora](changelog/202610180300-o-relato-de-um-filho-comeca-o-turno.md)
