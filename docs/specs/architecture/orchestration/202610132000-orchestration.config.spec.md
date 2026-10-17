# Config: Orquestrador — primeira fatia

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: flag > variável de ambiente > arquivo de config > default.
> Contrato técnico em `202610132000-orchestration.p.spec.md`.

> **Estado.** Nada daqui é lido ainda: a fatia está planejada, não construída. Cada chave entra em
> `KnownKeys` (`internal/config/toml.go`) **no mesmo PR** que passa a lê-la — o passo da `.i` que a
> nomeia —, e não antes: chave declarada que ninguém lê é o defeito que
> `TestEveryKeyTheSpecsDeclareIsReadSomewhere` existe para pegar. Por isso as tabelas abaixo abrem
> pela chave, não pela variável.

Todas as chaves são da seção `[orchestrator]` do `config.toml`, com a variável `DCODE_ORCHESTRATOR_*`
correspondente, bijetora como o resto do esquema. Listas são texto separado por vírgula, como
`sandbox.writable`. Durações aceitam a gramática de `time.ParseDuration`. Valor inválido é **erro
na abertura do daemon**, com a chave e o valor — nunca o default em silêncio.

## 1. Ligar, e onde mora

| Chave | Variável | Default | Validação | Passo |
|---|---|---|---|---|
| `orchestrator.enabled` | `DCODE_ORCHESTRATOR_ENABLED` | `false` | booleano | 5 |
| `orchestrator.dir` | `DCODE_ORCHESTRATOR_DIR` | `<state>/orchestrate` | absoluto; criado `0700`; fora de qualquer projeto que o daemon conduz | 4 |
| `orchestrator.closed_retention` | `DCODE_ORCHESTRATOR_CLOSED_RETENTION` | `1h` | de `0s` a `24h`; `0s` tira o fechado na hora | 6 |

`false` por padrão nesta fatia: a orquestradora é opt-in até a segunda fatia, e `POST /v1/projects`
responde `orchestrator_unavailable` dizendo a chave. `<state>` é o `Roots.State` de
`config.DiscoverRoots` (`DCODE_STATE_DIR`, XDG, ou `~/Library/Application Support/dcode` no macOS).
A orquestradora também exige `record.enabled`, já declarada em `configuration`.

`1h` é a decisão do dono em Q-E. O log de decisões guarda o fechado para sempre; a retenção é só da
lista.

## 2. Filhos e perfis por papel (RN-7, RN-8)

| Chave | Variável | Default | Validação | Passo |
|---|---|---|---|---|
| `orchestrator.max_children` | `DCODE_ORCHESTRATOR_MAX_CHILDREN` | `4` | inteiro de 1 a 16, e menor que `daemon` `MaxSessions` menos 2 | 6 |
| `orchestrator.max_returns` | `DCODE_ORCHESTRATOR_MAX_RETURNS` | `3` | inteiro de 1 a 10 | 8 |
| `orchestrator.profile` | `DCODE_ORCHESTRATOR_PROFILE` | vazio = `model.name` | nome de perfil de `models.toml`, ou vazio | 5 |
| `orchestrator.child_profiles` | `DCODE_ORCHESTRATOR_CHILD_PROFILES` | vazio = só o modelo da orquestradora | lista de perfis de `models.toml`; nome que não existe é erro | 6 |
| `orchestrator.reviewer_profiles` | `DCODE_ORCHESTRATOR_REVIEWER_PROFILES` | vazio = o primeiro de `child_profiles` | idem | 8 |
| `orchestrator.branch_prefix` | `DCODE_ORCHESTRATOR_BRANCH_PREFIX` | `dcode/` | passa em `git check-ref-format --branch` com um nome depois | 6 |
| `orchestrator.report_debounce` | `DCODE_ORCHESTRATOR_REPORT_DEBOUNCE` | `2s` | de `0s` a `1m` | 7 |
| `orchestrator.token_ceiling` | `DCODE_ORCHESTRATOR_TOKEN_CEILING` | `20000000` | inteiro de 100000 a 1000000000 | 6 |

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

O nome da branch é `<branch_prefix><slug do title>-<6 primeiros do id do filho>`.

## 3. Gate, revisão e merge (RN-5, RN-6)

| Chave | Variável | Default | Validação | Passo |
|---|---|---|---|---|
| `orchestrator.gate_command` | `DCODE_ORCHESTRATOR_GATE_COMMAND` | vazio = `verify.command` | texto; vazio com `verify.command` vazio é `gate_undeclared` no merge, não erro de abertura | 9 |
| `orchestrator.gate_timeout` | `DCODE_ORCHESTRATOR_GATE_TIMEOUT` | `30m` | de `1m` a `6h` | 9 |
| `orchestrator.review_max_diff_bytes` | `DCODE_ORCHESTRATOR_REVIEW_MAX_DIFF_BYTES` | `131072` | inteiro de 4096 a 1048576 | 8 |
| `orchestrator.forge` | `DCODE_ORCHESTRATOR_FORGE` | `auto` | `auto`, `github` ou `local` | 10 |
| `orchestrator.main_branch` | `DCODE_ORCHESTRATOR_MAIN_BRANCH` | vazio = a que `vcs` detecta | nome de branch que existe | 5 |
| `orchestrator.ci_timeout` | `DCODE_ORCHESTRATOR_CI_TIMEOUT` | `45m` | de `1m` a `6h` | 10 |
| `orchestrator.sensitive` | `DCODE_ORCHESTRATOR_SENSITIVE` | ver abaixo | lista de globs de `policy.Glob` | 10 |
| `orchestrator.sensitive_diff_lines` | `DCODE_ORCHESTRATOR_SENSITIVE_DIFF_LINES` | `800` | inteiro de 50 a 100000 | 10 |

`auto` é `github` quando o repositório tem `origin` e `gh auth status` responde; senão `local`, e a
primeira vez é dita (`forge.local`). `github` declarado sem `gh` é erro dito no `POST /v1/projects`,
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
