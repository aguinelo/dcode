# `/loop` como fachada sobre a RN-10

**Data:** 2026-08-25
**Specs afetadas:** nova família `202608252000-loop-command` (`.r`, `.p`,
`.config`, `.i`). Sem mudanças em outras specs.

> **Regra:** o dcode já tem um ciclo que executa contra uma `DoneSet` e para
> por progresso (`202608102100`). `/loop` é a forma de o usuário **declarar
> intenção** — "execute esta spec" — e o harness traduzir essa declaração na
> mesma `DoneSet` que `done.toml` já produz, lida de uma fonte diferente.

## O que isto é

Uma fachada. Não uma máquina nova. Não um ciclo novo. Não um `StopReason`
novo. Tudo o que o ciclo precisa para executar `/loop` **já existe** desde a
RN-10 (`202608102100`):

- `Criterion`, `DoneSet`, `CriterionState`, `Progressed` — em
  `internal/loop/done.go`.
- `StopIncomplete`, `MaxStallCycles`, `DoneEnabled` — em `Config`.
- O ciclo da RN-1 que consome a `DoneSet` e reentra por progresso.

O que falta é **alimentar a `DoneSet` a partir de uma pasta `specs/`**. O
caso de uso que motivou é a pipeline do Code Plain, onde 16 specs vivem em
`specs/2026-08-25-*/spec.md + tasks.md` com critérios verificáveis
declarados (cobertura, Lighthouse, smoke). Sem `/loop`, "implementar a spec
X" obriga o usuário a traduzir `tasks.md` em `done.toml` a cada vez. Com
`/loop`, o harness lê a pasta e produz a `DoneSet` direto.

## O desenho

### Origem da `DoneSet`, agora plugável

`internal/app/done.go` (`loadDoneSet`) lê `.dcode/done.toml` ou cai pro
`verifyCommand` legado. Esta mudança adiciona uma terceira origem — uma pasta
com `tasks.md` — sem mexer nas duas primeiras:

```
internal/loop/loopcommand/
    loopspec.go       # parser: tasks.md → []Criterion
    dispatch.go       # escolhe entre done.toml e LoopSpec
    session.go        # cria sessão dedicada, popula Config.Done
```

A separação é deliberada: o `app/` carrega `DoneSet`; o `loopcommand` **define**
`DoneSet` a partir de formato externo. Fronteira explícita entre as duas
preocupações.

### Parser é determinístico e total

`LoadSpec` **nunca** panica, **nunca** inventa, **nunca** deriva critério de
prosa. Três casos de borda, todos com saída explícita:

| Entrada | Saída |
|---|---|
| `tasks.md` com critérios verificáveis | `LoopSpec{Criteria: [...]}` |
| `tasks.md` com zero critérios explícitos | `LoopSpec{Criteria: nil}` |
| `tasks.md` malformado | `error` |

Prosa na spec ("smoke manual", "validar com o usuário") **não vira critério**.
Aparece no relatório final como `CriterionUnavailable` via mecânica já
existente: comando vazio em `Criterion` é `unavailable` (`done_test.go:104`).

### Reconhecimento no cliente

`/loop` é interceptado **antes** de virar entrada de turno. O texto do
comando **não** entra no histórico. Mesma regra da `agent-loop` §3 — sintaxe
no prefixo invalida o cache a cada turno (ADR-03).

### Sessão dedicada

`/loop` cria uma sessão nova a cada invocação, com `ID()` derivado do
basename da spec + timestamp. A sessão interativa fica intacta. Justificativa
completa na `loop-command.r §4.2`: misturar `DoneSet` quebra o isolamento de
progresso.

### `Protected` é declaração

Os caminhos que o agente não pode modificar em silêncio são declarados pelo
operador — no frontmatter de `tasks.md` (`protected = [...]`) ou via flag
`--protect`. **Não há default**. O harness não infere `Protected` da posição
da spec.

A forma exata (frontmatter vs. flag vs. ambos) é decisão do Code Plain,
não desta spec. A spec aceita o que vier; não prescreve.

## Fronteira de determinismo

| Parte | Regime |
|---|---|
| Reconhecimento do `/loop` | determinístico |
| Resolução do caminho | determinístico |
| Parsing de `tasks.md` | determinístico |
| Extração de `Protected` declarado | determinístico |
| Construção da `DoneSet` | determinístico |
| Criação de sessão dedicada | determinístico |
| Execução do turno contra a `DoneSet` | já existe (ciclo da RN-1) |
| Como o agente cumpre cada critério | já existe (mediado, limiar `fixes-cause-not-measure`) |

A linha inteira do que esta spec adiciona é determinística. A parte mediada
herda os limiares e invariantes da `agent-loop` — esta spec não toca neles.

## Contratos comportamentais novos

Quatro cenários, todos com fixture em `internal/evals/loop-command/`:

| ID | Cenário | Limiar |
|---|---|---|
| `loop-parses-spec` | parser extrai `DoneSet` correta de `tasks.md` bem-formado | ≥ 99% |
| `loop-ignores-prose` | prosa vira `unavailable`, não vira `Criterion` | ≥ 99% |
| `loop-protect-declared` | `protected` declarado é honrado | ≥ 95% |
| `loop-protect-absent` | sem declaração, nenhum path é protegido | ≥ 99% |

Os dois primeiros são o coração da RN-10 desta spec: o parser tem que extrair
correto, e prosa não pode virar critério. Os dois últimos materializam a
RN-4: declaração é a única forma de `Protected` entrar.

## O que esta mudança **não** faz

- **Não** muda o ciclo de execução. Tudo da `agent-loop` permanece intacto.
- **Não** muda o formato de `tasks.md` — esse formato é do Code Plain. Esta
  spec consome o que vier.
- **Não** escreve em `tasks.md` (marcar checkboxes). Ler é trabalho do
  parser; escrever é do agente, igual sempre foi.
- **Não** prescreve a forma de `Protected` (frontmatter, flag, ambos).
  Aceita o que o Code Plain declarar.
- **Não** adiciona `StopReason` novo. O ciclo usa o que já tem.

## Risco principal e o que o mitiga

O risco é o parser inventar critério onde há prosa — exatamente o que a
RN-10 da `agent-loop` proíbe. A mitigação está em duas camadas:

1. **Parser estrito.** Só vira `Criterion` o que casa o padrão `- [ ] N.
   \`path\` — desc. verify: \`cmd\``. Qualquer outra coisa é prosa ou erro.
2. **Teste obrigatório do Passo 1 da `.i`** com golden file da
   `tasks.md` real do Code Plain. Se o parser extrair algo errado, o
   golden quebra antes de qualquer fixture de eval rodar.

E o **teste mais importante** desta spec é o que verifica que `Progressed`
da `agent-loop` continua sendo chamado entre ciclos quando `DoneSet` veio
de `LoopSpec` em vez de `done.toml`. Se essa chamada for pulada por
engano (porque o dispatch introduziu um caminho de "turma nova"), o
`StopIncomplete` da RN-10 vira `StopDone` silenciosamente. Por isso o
teste é listado explicitamente em `loop-command.p §8` como invariante
verificável, e replicado em `loop-command.i §7`.

## Impacto

- Novo pacote `internal/loop/loopcommand/`. Sem mudança em `internal/loop/`
  raiz.
- Novo reconhecimento de input no client. Sem mudança no protocolo
  (`202608072240`).
- Novo evento `loop.session.created` (ou equivalente) para a sessão
  dedicada — extensão aditiva, não quebra consumidores.
- Limiares novos em `internal/evals/loop-command/`, atrás de build tag.
  Mesmo gate que a `agent-loop` já tem.

## Pendência de processo

A passagem de `experimental` para `stable` exige o **Passo 5 da `.i`**:
confirmação por issue/PR no **Code Plain** de que o formato `tasks.md`
está congelado. Sem isso, o parser pode falar com fantasma na próxima
mudança de spec do Code Plain, e o gate de promoção trava. Esta é a
única dependência cross-repo da spec.
