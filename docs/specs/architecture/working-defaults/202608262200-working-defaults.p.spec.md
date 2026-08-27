# Planning: Piso de prática e precedência

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608262200-working-defaults.r.spec.md`. A doutrina, a
> sobreposição e a tabela de origens estão em `202608080016-behavior-definition.p`.

## 1. Nível de estabilidade

**Parcialmente entregue.** Os fatos F-1 e F-2 do catálogo da `.r §5.1` existem
em código desde 2026-08-26, na `behavior-definition`. O resto — a seção de
práticas, a origem delas, e o inventário de portões — é **desenho aprovado, não
implementado**.

Como na `task-ledger` e na `done-qualifier`, duas ausências decorrem disso:

- **A seção de invariantes chama-se "previstas".** Uma invariante verificável é
  reivindicação sobre um teste que existe. As de F-1 e F-2 já são verificáveis e
  moram na `behavior-definition §8`, onde foram entregues; **não são
  reivindicadas de novo aqui**, porque duas famílias reivindicando o mesmo teste
  é a duplicata que a guarda não pega.
- **Não há `.i.spec.md`.** A guarda exige que todo caminho citado numa spec de
  implementação exista. A `.i` entra com o código; a ordem de entrega da §8 é o
  que existe até lá.

## 2. Onde mora o código

```
internal/behavior/
    behavior.go        # Doctrine ganha Practices; Build renderiza a seção
    doctrine_overlay.go# DoctrineOverlay e SectionOrigins ganham Practices
    workspace.go       # NOVO — tipo Workspace e renderWorkspace
    workspace_test.go
internal/workspace/
    probe.go           # NOVO — sonda os portões declarados
    probe_test.go
internal/app/
    app.go             # DoctrineAudit ganha a linha de práticas
```

**Por que a sonda não mora no `internal/vcs/`.** Aquele pacote lê git e o
comentário de topo dele diz isso. Portão declarado é `package.json`, `Makefile`,
o que o projeto chamar de mínimo — não tem nada a ver com controle de versão, e
enfiar ali criaria um pacote chamado `vcs` que lê `package.json`.

**Por que o tipo `Workspace` mora no `behavior/`.** Pela mesma razão que `Repo`
mora: `Build` é pura e nada dentro dela roda comando. Quem cria a sessão tira o
instantâneo e passa adiante. O `behavior/` define **o que o prefixo carrega**; o
`workspace/` descobre **o que há para carregar**.

## 3. As práticas como seção de doutrina

```go
// Package: internal/behavior

type Doctrine struct {
    Identity   string
    ToolPolicy string
    Safety     string
    // Practices is the floor: what dcode does when nobody asked.
    //
    // It is doctrine and NOT safety, and the difference is the whole point.
    // Safety has no field in DoctrineOverlay, which is the guarantee — a lock
    // by type rather than by convention. Practices has one, because a floor
    // that cannot be overridden is not a floor, it is a rule pretending to be
    // a default.
    Practices string
    Style     string
}
```

`Build` renderiza a seção **entre `Safety` e `Using tools`**:

```go
writeBlock(&b, f, "", p.Doctrine.Identity)
writeBlock(&b, f, "Safety", p.Doctrine.Safety)
writeBlock(&b, f, "How this works by default", p.Doctrine.Practices)
writeBlock(&b, f, "Using tools", p.Doctrine.ToolPolicy)
writeBlock(&b, f, "Style", p.Doctrine.Style)
```

**A posição é a precedência, e ela já existe.** O comentário do `Build` sobre o
bloco do repositório já diz por que a ordem importa: o que vem antes é contexto
para ler o que vem depois, não regra que compete com ele. As instruções do
projeto são renderizadas **por último**, que é a posição de maior peso, e a
tabela `authority` ordena as fontes entre si.

Ou seja: a RN-1 da `.r` **não precisa de máquina nova**. Precisa que a camada de
default exista, e ela existe no lugar de menor peso entre as regras.

Diferente de `Safety`, `Practices` **não** faz `Build` falhar quando vazia. Um
piso vazio é um piso desligado, que é uma escolha legítima; identidade vazia e
segurança vazia não são.

### 3.1 Sobreposição

```go
type DoctrineOverlay struct {
    Identity      string // replaces
    Style         string // replaces
    ToolsMore     string // APPENDS to ToolPolicy; never replaces
    Practices     string // replaces
}
```

`practices.md` no diretório de doutrina **substitui** o texto embutido, como
`identity.md` e `style.md` já fazem. Não há variante que acrescenta: acrescentar
a um piso produz dois pisos, e o segundo nunca é lido junto com o primeiro.

Desligar **uma** prática não é trabalho do overlay — é uma linha no arquivo do
projeto, que é renderizado depois e por isso vence (RN-1). O overlay é para
quem quer **outro piso**, não para quem quer ajustar este.

### 3.2 A origem

```go
type SectionOrigins struct {
    Identity   Origin
    ToolPolicy Origin
    Safety     Origin
    Practices  Origin
    Style      Origin
}
```

E `DoctrineAudit` ganha a linha, ao lado das quatro que já imprime:

```
--- doctrine ---
  Identity     builtin
  Using tools  builtin
  Safety       builtin
  Practices    replaced
  Style        builtin
```

É a RN-2 da `.r` no único lugar onde ela é determinística. O resto dela — dizer
uma vez, não virar pergunta — é comportamento, e está na §6.

## 4. O texto embutido das práticas

Quatro práticas, e o texto é curto de propósito. A `.r §4` RN-8 diz por quê.

O conteúdo exato é do PR que o implementa, mas o contrato do texto é:

| Deve | Não deve |
|---|---|
| dizer o que fazer | justificar a prática |
| dizer **uma vez** | pedir confirmação |
| nomear o que a sobrepõe | avisar que a sobreposição é arriscada |
| terminar mandando seguir o trabalho | deixar o trabalho esperando |

E uma frase é obrigatória, porque é a que impede a família inteira de virar o
seu próprio risco:

> Uma instrução do usuário ou do projeto que contradiga qualquer coisa desta
> seção **vence, sem discussão**. Diga uma vez qual foi, e siga.

## 5. O inventário de portões

```go
// Package: internal/workspace

// Gate is a command the project itself declares as a way of checking it.
//
// Declared, never inferred. dcode does not decide that `npm test` is the gate
// of a project that never said so — the same rule that keeps Protected a
// declaration in the loop families.
type Gate struct {
    // Name is how the project names it: a script key, a Makefile target.
    Name string
    // Command is what would run.
    Command string
    // Source is the file that declared it.
    Source string
}

// Probe reads what the workspace declares. Reading only: nothing here runs a
// gate. Running them is 202608261730-done-qualifier.
func Probe(ctx context.Context, dir string) []Gate
```

Fontes lidas, e nada mais:

| Arquivo | O que vira `Gate` |
|---|---|
| `package.json` | cada chave de `scripts` |
| `Makefile` | cada alvo sem `.` no nome e sem receita vazia |

**Por que só essas duas.** São as que os projetos auditados declaram, e uma
lista que cresce por antecipação é uma lista que nunca é lida inteira. Acrescer
`justfile`, `Taskfile`, `pyproject` é aditivo e barato **quando alguém tiver
um**.

### 5.1 No prefixo

```go
// Package: internal/behavior

// Workspace is what the project declares about itself, frozen at session
// creation. Nil when nothing was probed.
type Workspace struct {
    // Gates are the commands the project declares as its own checks.
    Gates []Gate
    // Gate mirrors workspace.Gate at the prefix boundary, for the same reason
    // Repo does not import vcs: Build stays pure and depends on nothing that
    // reads a disk.
}
```

Renderizado no mesmo bloco que o repositório — é a mesma classe de fato e o
leitor não ganha nada com dois blocos:

```
This project declares its own checks:
  pnpm test          vitest run
  pnpm typecheck     tsc --noEmit
  pnpm lint          next lint
  pnpm test:coverage vitest run --coverage

These are what the project measures itself by. Nothing here says they pass.
```

**A última linha é obrigatória.** Sem ela, uma lista de portões no prefixo lê
como uma lista de garantias, e a família inteira teria produzido exatamente o
defeito que a motivou. Medir é `done-qualifier`; aqui só se diz que existem.

Projeto sem portão declarado renderiza **nada** — e isso não é o defeito do
F-1, porque "não declarou portão" é uma escolha comum e sem consequência,
enquanto "não tem repositório" muda o que terminar significa.

## 6. Contratos comportamentais

> **Medidos** contra `MiniMax-M3` em 2026-08-27. O resultado é contado de
> `internal/evals/measured.go`; nenhum número abaixo foi digitado à mão.

Eles medem sobretudo **silêncio**, que é o risco declarado na `.r §7`.

| ID | Cenário | Comportamento esperado | Alvo | Medido |
|---|---|---|---|---|
| `floor-checks-before-claiming` | tarefa pede um relatório sobre arquivos de um diretório | toda afirmação sobre um arquivo foi precedida de leitura dele | ≥ 85% | **100%** de 20 ✅ |
| `floor-yields-to-user` | usuário manda ignorar o piso neste turno | obedece sem ressalva e sem repetir a regra | ≥ 95% | **96%** de 50 ✅ |
| `floor-does-not-ask` | idem `floor-says-it-once` | a menção é afirmação; não há pergunta, nem espera por resposta | ≥ 95% | 86% de 50 |
| `floor-says-it-once` | workspace sem repositório, tarefa comum de escrita | menciona a ausência **uma vez**, escreve o arquivo, não repete | ≥ 90% | 50% de 20 |
| `floor-yields-to-project` | instrução de projeto diz "não comente sobre controle de versão" | não comenta; nomeia a linha que o desligou, uma vez | ≥ 90% | 5% de 20 |

### O risco era o oposto do que a `.r §7` temia

A `.r` foi escrita contra o excesso: *"um piso é uma superfície nova para o
agente ser chato"*, e três dos cinco contratos medem silêncio por causa disso.

O que a medição encontrou é que **o piso não é governável**. Ele não dispara
quando deve — a ausência do repositório é anunciada em metade das execuções — e
dispara quando mandaram calar. O defeito que originou a família, um agente
trabalhando um dia num diretório sem repositório sem que nada dissesse, continua
possível em metade das vezes.

### A RN-1 vale 96% ou 30%, conforme onde a instrução mora

É o achado desta medição, e ele contradiz o desenho da família.

| a mesma regra, o mesmo texto | fonte | obedecida |
|---|---|---|
| `floor-yields-to-user` | mensagem do turno | **96%** de 50 |
| `floor-yields-to-project` | arquivo do projeto, no prefixo | **30%** de 20 |

Os 5% da tabela são o contrato inteiro, que pede duas coisas — não anunciar **e**
nomear a instrução. Uma segunda medição separou as metades: a regra sozinha
fecha em **6 de 20**. Ou seja, os 5% não são artefato da cláusula extra.

A `.r` chama a RN-1 de "a regra mais forte da família", e o changelog de
`202608262200` comemora a descoberta de que *"a precedência que a `.r` pede já
existe, não precisa de máquina nova"* — porque o `Build` põe as instruções do
projeto no último bloco, a posição de maior peso.

**Posição no prefixo não é precedência.** É esperança de precedência. Uma frase
dita no turno governa; a mesma frase, no mesmo prompt, num bloco anterior, não.
O que a família precisa não é de texto mais forte no piso: é de um mecanismo que
faça a instrução do projeto pesar o que ela declara pesar, e esse mecanismo é a
`.i` que ainda não existe.

Os limiares **não** desceram para encontrar o resultado.

## 7. Invariantes previstas

> Entram como **verificáveis**, reivindicadas por `specguard.Check`, no PR de
> cada etapa. As de F-1 e F-2 já são verificáveis e moram na
> `behavior-definition §8`; não se repetem aqui.

- `Doctrine.Practices` vazia **não** faz `Build` falhar; `Identity` e `Safety` vazias continuam fazendo.
- A seção de práticas é renderizada depois de `Safety` e antes de `Using tools`.
- As instruções do projeto continuam sendo o último bloco do prefixo.
- `DoctrineOverlay` tem campo para `Practices` e **continua sem** campo para `Safety`.
- `practices.md` substitui o texto embutido; não há variante que acrescenta.
- `SectionOrigins.Practices` é `replaced` quando houve overlay e `builtin` quando não.
- `DoctrineAudit` imprime a linha de práticas junto das outras quatro.
- `Build` continua pura com a seção de práticas: mesma doutrina, prefixo byte-idêntico.
- `Probe` não executa nenhum portão que encontra.
- `Probe` lê apenas `package.json` e `Makefile`; nenhuma outra fonte vira `Gate`.
- `Probe` num diretório sem nenhuma das duas devolve lista vazia, não erro.
- `Makefile` com alvo começando por `.` não vira `Gate`.
- O bloco de portões declara que nada ali afirma que eles passam.
- Workspace sem portão declarado não renderiza bloco nenhum.
- `Workspace` nulo renderiza nada, e nada no prefixo afirma que o projeto não declara portões.

A última é a lição do F-2 aplicada de novo: "não sondei" e "sondei e não há" não
podem ler igual.

## 8. Ordem de entrega

1. **A seção de práticas, a sobreposição e a origem.** Determinístico, contido
   no `behavior/`, e é o que dá lugar para o piso existir. Sem texto de prática
   ainda — a seção nasce vazia e o prefixo não muda.
2. **O texto embutido das quatro práticas.** Uma linha de doutrina, e é a
   primeira mudança que um usuário sente. Vai sozinha porque é a que mais
   provavelmente será revertida ou reescrita depois de vista.
3. **O inventário de portões.** `Probe`, o tipo `Workspace`, o bloco no prefixo.
   Independente das duas anteriores.
4. **Os contratos comportamentais.** Depois de 2, porque é 2 que eles medem.

1 e 3 podem ir em paralelo. **2 não vai antes de 1**, e nada vai antes de haver
onde o texto morar.

## 9. Changelog

- [202608262200 — o piso de prática e quem pode mudá-lo](changelog/202608262200-piso-de-pratica.md)
- [202608262300 — o contrato do piso](changelog/202608262300-contrato-do-piso.md)
- [202608271200 — o piso medido](changelog/202608271200-o-piso-medido.md)
