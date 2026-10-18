# Config: Orquestrador — primeira fatia

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: flag > variável de ambiente > arquivo de config > default.
> Contrato técnico em `202610132000-orchestration.p.spec.md`.

> **Estado.** `orchestrator.dir` é lida desde o passo 4; `enabled`, `profile` e `main_branch`, desde o
> passo 5; `gate_command` e `gate_timeout`, desde o passo 9 (`orchestrate.GateFrom`);
> `reviewer_profiles` e `review_max_diff_bytes`, desde o passo 8 (`orchestrate.ReviewFrom`);
> `max_children`, `child_profiles`, `branch_prefix` e `token_ceiling`, desde o passo 6
> (`orchestrate.ReadLimits`), e `closed_retention` com `dir`, onde o daemon nasce
> (`orchestrate.ClosedRetention`); `forge`, `ci_timeout`, `sensitive` e `sensitive_diff_lines`,
> desde o passo 10 (`orchestrate.MergeFrom`); `max_returns` e `report_debounce`, desde a fiação dos
> passos 7 a 10 (`orchestrate.ReadLimits`); o resto está planejado, não construído. Cada
> chave entra em `KnownKeys` (`internal/config/toml.go`) **no mesmo PR** que passa a lê-la — o passo
> da `.i` que a nomeia —, e não antes: chave declarada que ninguém lê é o defeito que
> `TestEveryKeyTheSpecsDeclareIsReadSomewhere` existe para pegar. Por isso as tabelas abaixo abrem
> pela chave, não pela variável — o guarda só lê a linha que abre pela variável —, e a chave lida
> passa à tabela "Lidas", que abre pela variável, no PR que a lê.

Todas as chaves são da seção `[orchestrator]` do `config.toml`, com a variável `DCODE_ORCHESTRATOR_*`
correspondente, bijetora como o resto do esquema. Listas são texto separado por vírgula, como
`sandbox.writable`. Durações aceitam a gramática de `time.ParseDuration`. Valor inválido é **erro
na abertura do daemon**, com a chave e o valor — nunca o default em silêncio.

## 1. Ligar, e onde mora

Lidas:

| Variável | Chave | Default | Validação | Passo |
|---|---|---|---|---|
| `DCODE_ORCHESTRATOR_DIR` | `orchestrator.dir` | `<state>/orchestrate` | absoluto; criado `0700`; fora de qualquer projeto que o daemon conduz | 4 |
| `DCODE_ORCHESTRATOR_ENABLED` | `orchestrator.enabled` | `false` | booleano | 5 |
| `DCODE_ORCHESTRATOR_CLOSED_RETENTION` | `orchestrator.closed_retention` | `1h` | de `0s` a `24h`; `0s` tira o fechado na hora | 6 |

`false` por padrão nesta fatia: a orquestradora é opt-in até a segunda fatia, e `POST /v1/projects`
responde `orchestrator_unavailable` dizendo a chave. `enabled` e `dir` são lidas onde o `dcode serve`
nasce, por `orchestratorOptions` (`cmd/dcode/serve.go`): é o daemon que conduz, e um diretório só tem
um daemon, que o trava (`daemon.lock`). O daemon embutido num cliente de terminal nunca conduz — morre
com ele — e diz isso no `orchestrator_unavailable`. `<state>` é o `Roots.State` de
`config.DiscoverRoots` (`DCODE_STATE_DIR`, XDG, ou `~/Library/Application Support/dcode` no macOS).
A orquestradora também exige `record.enabled`, já declarada em `configuration`.

`orchestrator.dir` é lida por `orchestrate.Dir` (`internal/orchestrate/dir.go`): relativa é erro
com a chave e o valor, nunca resolvida contra o diretório onde o daemon nasceu; sem raiz de estado
não há default, e isso também é erro. "Fora do projeto" é conferido quando o log de um projeto
abre (`OpenDecisionLog`), porque só ali se sabe a raiz dele.

`1h` é a decisão do dono em Q-E. O log de decisões guarda o fechado para sempre; a retenção é só da
lista.

## 2. Filhos e perfis por papel (RN-7, RN-8)

Lidas, na cadeia de configuração da raiz do projeto, quando ele abre:

| Variável | Chave | Default | Validação | Passo |
|---|---|---|---|---|
| `DCODE_ORCHESTRATOR_PROFILE` | `orchestrator.profile` | vazio = `model.name` | nome de perfil de `models.toml`, ou vazio | 5 |
| `DCODE_ORCHESTRATOR_MAIN_BRANCH` | `orchestrator.main_branch` | vazio = a que `vcs` detecta | nome de branch que existe | 5 |
| `DCODE_ORCHESTRATOR_MAX_CHILDREN` | `orchestrator.max_children` | `4` | inteiro de 1 a 16, e menor que `daemon` `MaxSessions` menos 2 | 6 |
| `DCODE_ORCHESTRATOR_CHILD_PROFILES` | `orchestrator.child_profiles` | vazio = só o modelo da orquestradora | lista de perfis de `models.toml`; nome que não existe recusa o filho com `model_not_allowed`, dito | 6 |
| `DCODE_ORCHESTRATOR_BRANCH_PREFIX` | `orchestrator.branch_prefix` | `dcode/` | passa em `git check-ref-format --branch` com um nome depois | 6 |
| `DCODE_ORCHESTRATOR_TOKEN_CEILING` | `orchestrator.token_ceiling` | `20000000` | inteiro de 100000 a 1000000000 | 6 |
| `DCODE_ORCHESTRATOR_REVIEWER_PROFILES` | `orchestrator.reviewer_profiles` | vazio = o primeiro de `child_profiles` | lista de perfis de `models.toml`; nome que não existe é erro | 8 |
| `DCODE_ORCHESTRATOR_MAX_RETURNS` | `orchestrator.max_returns` | `3` | inteiro de 1 a 10 | 8 |
| `DCODE_ORCHESTRATOR_REPORT_DEBOUNCE` | `orchestrator.report_debounce` | `2s` | de `0s` a `1m` | 7 |

Valor inválido nestas é recusado no `POST /v1/projects` (`workspace_invalid`), com a chave e o
valor: só ali se sabe a raiz cuja cadeia as declara.

`max_children` conta filhos **vivos** (não terminais) do projeto, e o lote que passaria dele é
recusado inteiro. Quatro porque é o que o dia à mão de 2026-10-16 sustentou com uma pessoa lendo os
PRs; cada filho é uma sessão, e as duas sessões descontadas de `MaxSessions` são a orquestradora e um
revisor.

Os perfis são os de hoje (`models.toml`, `config.Profile`): a lista por papel é a **lista
permitida** da RN-8, e o `model` de um item do lote que não está nela é `model_not_allowed`. A
escalada para um perfil mais forte é do modelo, dentro da lista, e vai ao log com o porquê.

`token_ceiling` é o teto de tokens da RN-3, trazido para esta fatia pelo dono em 2026-10-17: lido
no passo 6, conferido no `delegate` contra `State.Spent` (input, output e cache lidos de toda sessão
do projeto). Atingido, o despacho para com `ceiling_reached`, dito na conversa e no log, e uma
pergunta oferece `continue` (zera a janela) ou `stop`. Vinte milhões são a ordem de uma noite de
quatro filhos no modelo de hoje; é um freio, não um orçamento. Os tetos de custo e de tempo, e
parar filho vivo com `budget`, ficam na segunda fatia (§5).

O nome da branch é `<branch_prefix><slug do title>-<6 últimos do id do filho>`: os seis primeiros são o
relógio do id, e se repetem entre os filhos de um lote.

## 3. Gate, revisão e merge (RN-5, RN-6)

Lidas:

| Variável | Chave | Default | Validação | Passo |
|---|---|---|---|---|
| `DCODE_ORCHESTRATOR_GATE_COMMAND` | `orchestrator.gate_command` | vazio = `verify.command` | texto; vazio com `verify.command` vazio é `gate_undeclared` na medição, não erro de abertura | 9 |
| `DCODE_ORCHESTRATOR_GATE_TIMEOUT` | `orchestrator.gate_timeout` | `30m` | de `1m` a `6h` | 9 |
| `DCODE_ORCHESTRATOR_REVIEW_MAX_DIFF_BYTES` | `orchestrator.review_max_diff_bytes` | `131072` | inteiro de 4096 a 1048576; diff acima é cortado inteiro e dito, nunca truncado | 8 |
| `DCODE_ORCHESTRATOR_FORGE` | `orchestrator.forge` | `auto` | `auto`, `github` ou `local` | 10 |
| `DCODE_ORCHESTRATOR_CI_TIMEOUT` | `orchestrator.ci_timeout` | `45m` | de `1m` a `6h` | 10 |
| `DCODE_ORCHESTRATOR_SENSITIVE` | `orchestrator.sensitive` | ver abaixo | lista de globs de `policy.Glob` | 10 |
| `DCODE_ORCHESTRATOR_SENSITIVE_DIFF_LINES` | `orchestrator.sensitive_diff_lines` | `800` | inteiro de 50 a 100000 | 10 |

`auto` é `github` quando o repositório tem `origin` e `gh auth status` responde; senão `local`, e a
primeira vez é dita (`forge.local`). `github` declarado sem `gh` é erro dito quando o passo de merge abre (o primeiro filho medido),
nunca queda silenciosa para `local`.

`gate_timeout` de 30 minutos cobre o `make check` deste repositório (lint, race, coverage, build)
com folga; gate que passa dele é `timeout`, que não é verde.

**O default de `orchestrator.sensitive`** (Q-E): `.github/**`, `scripts/**`, `Makefile`, `go.mod`,
`go.sum`, `**/package.json`, `**/package-lock.json`, `.dcode/**`, `**/*.r.spec.md`. Nesta fatia
"sensível" só marca a pergunta de merge (`Question.Sensitive`), porque em passo a passo todo merge já
pergunta; na segunda é o que faz o supervisionado parar. O veredito do revisor com ressalva e a
dependência nova, as outras duas metades de Q-E, entram com o supervisionado.

## 4. Constantes não configuráveis

| Constante | Valor | Motivo |
|---|---|---|
| Orquestradoras por projeto | 1 | RN-1: uma entrada só. |
| Merges simultâneos por projeto | 1 | RN-6: cada merge medido sobre a main de agora. |
| Nível de autonomia | `step` (passo a passo) | segunda fatia; nada nesta lê outro valor. |
| Rebase, `push --force`, `commit --amend` | nunca | RN-6; nenhum caminho de código os tem. |
| Prazo da aprovação de filho | nenhum (`ExpiresAt` zero) | Q-I: espera a pessoa. |
| Quem responde pergunta e aprovação de filho | só a pessoa | ADR-02. |
| Quem faz commit, merge e push do filho | o harness | o filho não grava no `.git` comum; o sandbox e a negação dizem o porquê. |
| `git add` | caminhos nomeados, nunca `-A` | o que entra é o que foi conferido contra o `owns`. |
| Ordem de uma transição | log → `state.json` → fio | o disco nunca à frente do log. |
| Versão do formato do log | `1` | `v` em cada linha. |
| Tamanho máximo de uma linha do log | 64 KiB | o campo maior é cortado e diz quanto. |
| Saída do gate no evento | os últimos 4 KiB | a inteira fica em `gates/`. |
| `Project.ID` | 24 hex de sha256 da raiz resolvida | estável entre reinícios e entre caminhos para a mesma árvore. |
| Texto do relato ao modelo | inglês | texto que o modelo lê é superfície de comportamento (`docs/conventions/LANGUAGE.md`, abertura). |
| Revisões sem veredito antes de parar | 2 | uma repetição; a segunda falta vai à orquestradora. |

## 5. Declaradas para a segunda fatia — não lidas nesta

Escritas aqui para que a forma não seja inventada depois; **não** entram em `KnownKeys` nesta fatia,
e nenhum passo da `.i` as lê.

| Chave | Default proposto | Regra |
|---|---|---|
| `orchestrator.autonomy` | `step` | RN-3: `step`, `supervised`, `autonomous`; padrão por projeto |
| `orchestrator.cost_ceiling` | `10` (dólares) | RN-3; pede a tabela de preços, que entra junto, e o custo em dólares na lista |
| `orchestrator.time_ceiling` | `6h` | RN-3 |
| `orchestrator.deploy_command`, `orchestrator.deploy_env`, `orchestrator.deploy_when` | vazios | RN-13 |
| `orchestrator.notify` | `true` | RN-10 |

Os limiares dos contratos com modelo real e o modelo contra o qual foram medidos entram com eles, na
segunda fatia, pelas chaves `eval.*` de `202608072334-provider-adapter.config.spec.md` §4.

## 6. Changelog

- [202610172200 — A primeira fatia, planejada](changelog/202610172200-a-primeira-fatia-planejada.md)
- [202610172330 — Toda decisão vai ao log com o porquê](changelog/202610172330-toda-decisao-vai-ao-log.md)
- [202610180000 — Um projeto tem uma orquestradora, que sobrevive ao daemon](changelog/202610180000-uma-orquestradora-por-projeto.md)
- [202610180100 — Um revisor lê o diff de cada filho](changelog/202610180100-um-revisor-le-o-diff-de-cada-filho.md)
- [202610180200 — Um filho trabalha na própria worktree](changelog/202610180200-um-filho-trabalha-na-propria-worktree.md)
- [202610180400 — O relato de um filho começa o turno da orquestradora](changelog/202610180400-o-relato-de-um-filho-comeca-o-turno.md)
