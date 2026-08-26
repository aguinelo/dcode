# `/loop` como fachada sobre a RN-10

**Data:** 2026-08-25, revisado em 2026-08-26
**Specs afetadas:** nova família `202608252000-loop-command` (`.r`, `.p`,
`.config`, `.i`). Sem mudanças em outras specs.

> **O que esta entrada descreve, e o que ela não descreve.** O desenho abaixo é
> o da família inteira. O que **existe em código** é o parser, o dispatch entre
> fontes e o `loop.Config` da sessão dedicada. `/loop` **não é digitável**: o
> reconhecimento no cliente é o Passo 3 da `.i` e não foi construído. A revisão
> de 2026-08-26 está na segunda metade deste arquivo.

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
completa na RN-2 do `.r`: misturar `DoneSet` quebra o isolamento de
progresso.

### `Protected` é declaração

Os caminhos que o agente não pode modificar em silêncio são declarados pelo
operador — no frontmatter de `tasks.md` (`protected = [...]`) ou via flag
`--protect`. As duas fontes são **união**, sem repetição e sem precedência.
**Não há default**. O harness não infere `Protected` da posição da spec.

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

## Contratos: quatro IDs, nenhum medido contra modelo

| ID | Cenário | Limiar |
|---|---|---|
| `loop-parses-spec` | `DoneSet` correta de `tasks.md` bem-formado | **100%**, asserção |
| `loop-ignores-prose` | prosa não vira `Criterion`; arquivo ilegível é erro | **100%**, asserção |
| `loop-protect-declared` | arquivo e argumento se unem | **100%**, asserção |
| `loop-protect-absent` | sem declaração, nenhum path é protegido | **100%**, asserção |

Os quatro descrevem o **parser**, que é determinístico. Medir contra modelo
gastaria vinte chamadas para imprimir `MET` num número que uma asserção já
decidiu — um verde de graça, que ninguém olha duas vezes.

Contrato mediado volta quando o Passo 3 existir e o modelo puder alcançar
`/loop`. Aí há comportamento a medir, e o número entra aqui com modelo e data.

## O que esta mudança **não** faz

- **Não** muda o ciclo de execução. Tudo da `agent-loop` permanece intacto.
- **Não** muda o formato de `tasks.md` — esse formato é do Code Plain. Esta
  spec consome o que vier.
- **Não** escreve em `tasks.md` (marcar checkboxes). Ler é trabalho do
  parser; escrever é do agente, igual sempre foi.
- **Não** prescreve a forma de `Protected` (frontmatter, flag, ambos).
  Aceita o que o Code Plain declarar.
- **Não** adiciona `StopReason` novo. O ciclo usa o que já tem.

## Risco principal e o que o mitiga (escrito antes do código)

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


---

# Revisão de 2026-08-26

O código chegou depois deste changelog, e não era o código que ele descreve.
Esta seção é a diferença, escrita para que a diferença não precise ser
redescoberta.

## O que a doc afirmava e não era verdade

- O `CHANGELOG.md` do produto dizia que `/loop` "runs the existing turn loop
  against it in a dedicated session". Nada chamava o pacote: `/loop` nunca foi
  digitável.
- O `ROADMAP.md`, **no mesmo commit**, dizia "specs are written, no code yet" —
  e o commit trazia ~700 linhas de código. Dois documentos do mesmo PR se
  contradizendo, nenhum dos dois certo.
- O `.config §4` dizia que os limiares "foram medidos contra o modelo
  declarado". Nenhuma medição jamais rodou, e as fixtures diziam "placeholder".
  A fixture era honesta e o `.config` não — e o `.config` é onde se confia.
- A `.p §5` declarava `NewSession(ctx, srv, opts) (SessionHandle, error)`, com
  `ID()` e `SubmitTurn()`. O código tinha `NewSession(opts) (loop.Config,
  string)`: nenhum servidor, nenhum handle, nenhuma sessão. Cinco invariantes
  da `.p §8` descreviam essa máquina inexistente, e nenhuma tinha teste.
- A `.p §3.2` dizia que o `protected` do arquivo "vence" o do argumento; o
  código unia os dois, e o comentário do código dizia o contrário da spec.
- `verify.toml`, citado na justificativa da RN-2, não existe em lugar nenhum
  do repositório.

## Três defeitos que a doc não conhecia

1. **O parser exigia travessão literal.** `- [ ] 1. \`a.ts\` - desc. verify:
   \`make test\`` — com hífen — devolvia zero critérios. Somado ao defeito 2,
   um `tasks.md` inteiro escrito com hífen virava "sem definição de pronto",
   que o ciclo relata como **pronto**. A feature estava a uma tecla de um verde
   falso.
2. **Linha que não casa virava silêncio.** O parser fazia `continue` em tudo
   que não reconhecia, então um arquivo de lixo voltava com `err == nil` e zero
   critérios — exatamente o que a RN-6 proíbe e o que a US-5 diz que o operador
   não pode receber.
3. **`os.IsNotExist` sobre erro embrulhado com `%w`.** Ele não segue a cadeia.
   O fall-through de `SourceAuto` para o comando legado era código inalcançável
   sob um comentário dizendo que era alcançável — e a única linha do gate de
   cobertura que reprovava apontava exatamente para ele.

## O teste que não testava

`TestInvariantsClaimed` e `TestLoopCommandInvariantsClaimed` montavam um mapa
que nunca liam e comparavam `const family = "loop-command"` consigo mesmo.
Existiam para o `strings.Contains` do specguard achar o nome da família num
arquivo, e não podiam falhar. Estavam duplicados em dois lugares, um em
português e outro em inglês, e um dos dois nem era alcançado pelo glob do
guard.

Substituídos por `specguard.Check` de verdade, que lê cada linha da `.p §8` e
reprova quando o teste nomeado não existe. Verificado nos dois sentidos:
renomear um teste reprova, tirar um nome do mapa reprova.

**A lição, e ela é sobre harness.** Todo guard com dente pegou alguma coisa: o
de nomes exportados forçou a isenção que prova que nada chama o pacote; o de
wiring forçou as quatro chaves `loop.*` a serem lidas por nome; o gate de
cobertura por pacote reprovou apontando para as linhas do defeito 3; o de
fixture forçou o material a existir, e foi ao escrevê-lo que a honestidade
apareceu. O único derrotado foi o que verifica por `strings.Contains` de um
nome num arquivo. Ele pediu uma string e recebeu uma string. **Guard que uma
string satisfaz é guard que uma string vai satisfazer.**

## O que mudou no código

| Antes | Depois |
|---|---|
| separador `—` obrigatório | só `- [ ] N.` e `verify: \`cmd\`` são sintaxe |
| linha desconhecida ignorada | sem nenhuma linha de tarefa é erro |
| `---` em qualquer posição abria frontmatter | só na linha 1; no meio é régua do markdown |
| `verify:` sem crases, cauda `exit` ilegível, `N` repetido: silêncio | erro nomeando a linha |
| `os.IsNotExist` | `errors.Is(err, fs.ErrNotExist)` |
| `NewSession`, que não criava sessão | `SessionConfig`, que monta o `Config` |
| 87.9% de cobertura, abaixo do piso | 97.6% |
