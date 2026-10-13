# Proposta: o `/loop` com definição de pronto no desktop

> **Estado: proposta, à espera do visto do dono.** Nada aqui foi construído.
> Cada passo da §8 é um PR de um tema só; os que mexem no núcleo levam a
> mudança de spec (changelog na família) antes do código, como
> `docs/conventions/SDD-HARNESS.md` pede. Onde esta proposta e uma spec
> divergirem, a spec vence.

**Por que este arquivo mora em `desktop/docs/proposals/` e não em
`docs/specs/`.** A regra do desktop (`desktop/AGENTS.md`, e o topo de
`docs/DECISIONS.md`) é que decisão de interface mora no desktop, sem spec SDD,
e que o desktop **não muda o protocolo**: o que ele precisa vira pedido ao
núcleo, com spec lá. A parte de núcleo deste desenho já é um pedido aceito — o
**N4** ("o `/loop` no daemon", pedido por **D24**) — e cai em famílias que já
existem (`loop-command`, `done-qualifier`, `client-server-protocol`). Família
existente não ganha um segundo `.r`: muda por changelog, escrito no PR que faz
a mudança. Então a proposta que o dono aprova é do desktop, e os changelogs de
spec nascem nos passos 1–5 da §8.

---

## 1. Problema

O `/loop` existe e funciona — na TUI. A sequência inteira mora no processo da
TUI (`internal/tui/program.go`, `loopOne`, `loopEverySpec`, `nextSpec`,
`commitProposal`, e o despacho no `case eventMsg`):

1. pergunta ao daemon o que cada pasta declara (`GET /v1/specs?measure=false`);
2. pasta sem critério abre uma sessão **qualificadora** (`Qualify`, modo plan);
3. o turno qualificador termina → `POST /v1/sessions/{id}/done` mede e escreve
   o `done.toml` → **o laço para** para a pessoa revisar o arquivo;
4. pasta com critério abre uma sessão de trabalho (`LoopSpec`, `Protect`), e o
   turno re-entra enquanto os critérios encolhem (`agent-loop` RN-10);
5. com um objetivo, repete por pasta pendente e termina com o veredito.

Três consequências:

- **O desktop não começa nem conduz um loop.** Ele desenha o selo e as luzes
  dos critérios (`state/pane.ts`, D13, D30) de um turno que já terminou, mas não
  tem como abrir um, nem mostrar em que pasta, em que fase ou em que ciclo está.
- **Dois clientes veriam dois loops diferentes.** A fila, a pasta corrente e o
  veredito vivem na memória da TUI. Um desktop que reimplementasse a sequência
  seria uma segunda cópia da regra — exatamente o que o D24 recusou.
- **O ciclo é invisível.** A re-entrada por critério acontece dentro do turno
  (`loop.Engine.checkDone`) e não emite nada: o cliente só sabe o estado dos
  critérios no `turn.completed`. "Ciclo 3, 2 de 5 critérios" não existe no fio.

E há **superfície morta** no protocolo: `done.proposed`, `done.signed` e
`SignDoneRequest` (`internal/protocol/protocol.go`) não têm emissor nem rota.
Sobraram do desenho em que a assinatura era um ida-e-volta dentro do turno; o
changelog `done-qualifier/202608270700-o-laco-le-as-ferramentas.md` trocou isso
pelo arquivo como superfície de revisão, e o resto ficou. `internal/specguard`
os mantém vivos com um motivo escrito ("nothing emits done.proposed yet"), o
`.config` da família declara `DCODE_QUALIFIER_MAX_SIGN_ROUNDS` e
`DCODE_QUALIFIER_SIGN_TIMEOUT` que ninguém lê, e o desktop decodifica os dois
eventos (`src/protocol/events.ts`). É promessa num fio versionado que nenhum
código cumpre.

## 2. Objetivos e não-objetivos

**Objetivos**

- O desktop começa um loop — por pasta, por objetivo, com `--protect` — com a
  mesma gramática da TUI, e os dois clientes veem **o mesmo** loop.
- A sequência do `/loop` sai da TUI para o daemon (N4), sem mudar o que ela faz.
- Um evento por ciclo de conferência dentro do turno, com o resultado de cada
  critério (N4, D24): passou, não passou, não pôde rodar — sem contagem de
  testes.
- A proposta da qualificadora aparece como revisão estruturada (classe de cada
  critério, a discordância entre o que o modelo disse e o que a medição achou),
  e o trabalho só começa quando a pessoa diz.
- Parar: agora, ou depois da pasta corrente.
- A superfície morta é resolvida — religada ou apagada (Q1).

**Não-objetivos**

- O loop de `refs/design/desktop/LOOP.md` (iteração como turno inteiro,
  orçamentos `--tokens/--tempo`, contagem por teste, `dcode loop`). Recusado no
  D24; continua recusado.
- Editar critérios dentro do desktop. O `done.toml` continua sendo a superfície
  de revisão (Q3).
- Loop sem pessoa para revisar. A qualificação é interativa
  (`done-qualifier` RN-6), e isso não muda.
- Vários loops no mesmo workspace ao mesmo tempo (Q7).

## 3. Fluxo de quem usa

### 3.1 Começar

Duas portas, uma gramática:

- **No campo de qualquer janela:** `/loop specs/home` ou `/loop implemente as
  specs pendentes --protect tests/**`. O desktop reconhece `/loop` como reconhece
  `/undo` (D40) — nunca vai ao modelo como mensagem (`loop-command` RN-3) — e o
  argumento segue para o daemon como foi digitado.
- **No "Nova sessão" (D31/D38):** um item "Loop…" abre a folha da Figura 1, que
  lista as pastas que o daemon conhece (`GET /v1/specs?measure=false`).

```
Figura 1 — Grade: a folha "Começar loop", dentro da janela que perguntou onde
┌─ dcode · ~/w/codeplain ─────────────────────────────────────────── × ┐
│ Começar loop                                                         │
│                                                                      │
│  ( ) Uma pasta      ( ) Um objetivo                                  │
│                                                                      │
│  Pastas                               critérios declarados           │
│  ○ specs/2026-08-25-home-page          4                             │
│  ○ specs/2026-08-27-checkout           0  · qualifica antes          │
│  ○ specs/2026-09-02-search             2                             │
│                                                                      │
│  Objetivo  [ implemente todas as specs pendentes                  ]  │
│  Protegido [ tests/** ] [+]   o done.toml da pasta soma, nunca tira  │
│  Modelo  sonnet ▾     Modo  auto ▾                                   │
│                                                                      │
│                                      [ Cancelar ]  [ Começar  ⌘↵ ]   │
└──────────────────────────────────────────────────────────────────────┘
```

"0 · qualifica antes" é o que o daemon declara, não uma previsão do cliente.
Com um objetivo, o daemon mede antes (`measure=true`) e a janela mostra o plano
— quais pastas estão pendentes — antes de qualquer sessão abrir.

### 3.2 A janela do loop

**Um loop ocupa uma janela da grade, e a janela segue a sessão corrente do
loop** (Q5). O topo da janela ganha a régua do loop: a pasta, a fase, o ciclo,
e as luzes dos critérios atualizadas a cada conferência (`done.checked`), não
só no fim do turno.

```
Figura 2 — Grade: um loop trabalhando (janela 2 de 4)
┌──────────────────────────────────┬───────────────────────────────────┐
│ ↻ implemente as specs pendentes  │ ◆ refactor do parser          esp. │
│ spec 2 de 3 · specs/…-checkout   │ Pede rede · npm install            │
│ trabalhando · ciclo 3 · 12m04s   │ [1 permitir] [2 sessão] [3 negar]  │
│ ●●○○  2 de 4  · parado 1 de 3    │                                    │
│ ✓ unit  ✓ lint  ✗ e2e  ✗ a11y    │                                    │
│──────────────────────────────────│                                    │
│ … fluxo da sessão corrente …     │                                    │
│ ⟲ ciclo 2 voltou: e2e quebrou,   │                                    │
│   desfeito src/cart.ts           │                                    │
│                                  │                                    │
│ [Escreva para redirecionar…   ]  │ [Responda à aprovação acima     ]  │
│ ⏹ parar agora · ⏭ parar depois   │                                    │
├──────────────────────────────────┼───────────────────────────────────┤
│ …                                │ …                                  │
└──────────────────────────────────┴───────────────────────────────────┘
```

- **O passo-a-passo das pastas** abre ao clicar em "spec 2 de 3": cada pasta
  com a fase (`pendente`, `qualificando`, `proposta`, `trabalhando`, `pronta`,
  `ficou por fazer`) e a sessão dela; clicar abre a conversa daquela pasta numa
  janela própria, só leitura se terminou (D42).
- **"parado 1 de 3"** é o contador de ciclos sem progresso contra o teto
  (`MaxStallCycles`), que vem no evento. Quando a conferência volta atrás e o
  laço desfaz o ciclo (`undoCycle`), o fluxo diz o que voltou.
- **Escrever** durante o turno redireciona (D23). **Esperando aprovação**, a
  janela vai para o topo da atenção (D30), como qualquer outra.
- **Parar agora** interrompe o turno e encerra o loop; **parar depois** deixa a
  pasta corrente terminar e não abre a próxima.

### 3.3 A revisão da proposta

Pasta sem critério passa pela qualificadora. Quando o turno dela termina, o
daemon mede e escreve o `done.toml` — como hoje — e o loop fica em
`awaiting_review`. A janela mostra a proposta como o daemon a mediu:

```
Figura 3 — Grade (maximizada): a proposta esperando revisão
┌─ ↻ specs/2026-08-27-checkout · proposta escrita ────────────────────┐
│ specs/2026-08-27-checkout/done.toml · 3 critérios · medidos agora    │
│                                                                      │
│  aceitação  ✗ exit 1   checkout total      go test ./cart -run Total │
│  regressão  ✓ exit 0   suíte inteira       go test ./...             │
│  regressão  ✓ exit 0   sem arg sai ≠ 0     bash pay.sh; test $? -ne 0│
│     ⚠ o proponente disse que falharia, e passou — confira o porquê   │
│                                                                      │
│  Protegido: tests/**, specs/**/done.toml                             │
│                                                                      │
│  O trabalho começa quando você disser. Edite o arquivo, se quiser:   │
│  o daemon mede de novo antes de começar.                             │
│                                                                      │
│     [ Abrir o arquivo ]   [ Parar o loop ]   [ Começar o trabalho ]  │
└──────────────────────────────────────────────────────────────────────┘
```

- A linha com ⚠ é o `Mismatch` do `DoneCriterion`: onde o olho deve cair
  (`done-qualifier` RN-2, RN-3). Um critério **quebrado** (comando que não
  existe) vem marcado como tal, nunca como aceitação.
- "Abrir o arquivo" pede ao processo principal que abra o caminho no editor do
  sistema (`shell.openPath`), **só** quando o daemon é local; com daemon remoto,
  mostra o caminho para copiar. O renderer continua sem tocar o disco (D2).
- "Começar o trabalho" é `POST /v1/loops/{id}/continue`. Não há prazo que
  começa sozinho (`done-qualifier` RN-6).
- Num objetivo sobre um backlog, o daemon qualifica todas as pastas sem
  critério numa passada e para com os arquivos escritos — "uma sentada para
  revisar tudo" (o que a TUI já faz). A Figura 3 vira uma lista, uma proposta por
  pasta, e um "Começar o trabalho" para todas.

### 3.4 Crew

Na Crew (D33), o loop aparece na lista da esquerda como uma linha com `↻` e a
pasta corrente, e a aba **Pronto** da direita ganha o loop:

```
Figura 4 — Crew: aba Pronto de uma conversa em loop
┌ Painel ┬ Ativas ───────────────┬ conversa ───────────────┬ Mudanças Pronto Sessão ┐
│ ▣      │ ↻ specs pendentes 2/3 │ … fluxo …               │ Loop                   │
│ ◷      │   ◆ refactor parser   │                         │  objetivo: implemente… │
│ ◎      │   ● docs do site      │                         │  ✓ home-page   pronta  │
│ ⚙      │ dcode                 │                         │  ↻ checkout    ciclo 3 │
│        │   …                   │                         │  ○ search      pendente│
│        │                       │                         │ Critérios (ciclo 3)    │
│        │                       │                         │  ✓ unit  ✓ lint        │
│        │                       │                         │  ✗ e2e   ✗ a11y        │
│        │                       │                         │  sem progresso: 1 de 3 │
│        │                       │ [Escreva…             ] │ [⏭ depois] [⏹ agora]   │
└────────┴───────────────────────┴─────────────────────────┴────────────────────────┘
```

### 3.5 O fim

O daemon termina o loop com o veredito que a TUI já calcula (`loopVerdict`):
pastas trabalhadas, critérios met/total somados, os não cumpridos e os não
conferidos por nome, e as pastas que ficaram por fazer. A janela fica com ele no
topo; a conversa vira "vista" como qualquer outra (D30).

## 4. Mudanças no protocolo

Todas em `client-server-protocol`, nível `experimental`, abaixo de 1.0:
**MINOR**. Os tipos do desktop saem do `tygo` (D3); evento novo reprova os
testes do desktop até `events.ts` ler (é o que se quer).

### 4.1 O loop como recurso do daemon (N4)

| Método | Rota | Corpo → resposta |
|---|---|---|
| `POST` | `/v1/loops` | `StartLoopRequest` → `LoopRun` |
| `GET` | `/v1/loops?workspace=` | → `ListLoopsResponse` |
| `GET` | `/v1/loops/events` | **SSE**: `snapshot`, `changed`, `removed`, como `/conversations/events` |
| `POST` | `/v1/loops/{id}/continue` | — → `LoopRun` (sai de `awaiting_review`) |
| `POST` | `/v1/loops/{id}/stop` | `StopLoopRequest{when: "now" \| "after_spec"}` → `LoopRun` |

```go
type StartLoopRequest struct {
    Workspace   string   `json:"workspace"`          // absoluto
    Args        string   `json:"args"`               // o que veio depois de /loop, como digitado
    Model       string   `json:"model,omitempty"`
    SandboxMode string   `json:"sandbox_mode,omitempty"`
    Mode        string   `json:"mode,omitempty"`
}

type LoopRun struct {
    ID        string      `json:"id"`
    Workspace string      `json:"workspace"`
    Goal      string      `json:"goal,omitempty"`
    Protect   []string    `json:"protect,omitempty"`
    State     string      `json:"state"` // surveying | qualifying | awaiting_review | working | finished | stopped | failed
    Specs     []LoopSpecRun `json:"specs"`
    Current   int         `json:"current"`              // índice em Specs; -1 fora de pasta
    SessionID string      `json:"session_id,omitempty"` // a sessão corrente
    Verdict   *LoopVerdict `json:"verdict,omitempty"`
    Error     string      `json:"error,omitempty"`
    Started   time.Time   `json:"started"`
    Finished  time.Time   `json:"finished,omitempty"`
}

type LoopSpecRun struct {
    Path      string      `json:"path"`
    Phase     string      `json:"phase"` // pending | qualifying | proposed | working | done | unfinished | skipped
    SessionID string      `json:"session_id,omitempty"`
    Proposal  string      `json:"proposal,omitempty"` // caminho do done.toml escrito
    Standing  *Completion `json:"standing,omitempty"` // o último Completion desta pasta
}

type LoopVerdict struct {
    Worked    int      `json:"worked"`
    Met       int      `json:"met"`
    Total     int      `json:"total"`
    Unmet     []string `json:"unmet,omitempty"`
    Unchecked []string `json:"unchecked,omitempty"`
    Unfinished []string `json:"unfinished,omitempty"`
}
```

- **`Args` como digitado**, e o daemon faz o `ParseLoopArgs`. A TUI e o desktop
  reconhecem `/loop` no cliente (RN-3: a sintaxe nunca vira turno), mas a
  gramática passa a ter uma casa só. Hoje ela é Go dentro de `internal/tui`; ela
  muda para `internal/loop/loopcommand`, que já é a casa do resto.
- `Session` e `Conversation` ganham `loop_id,omitempty`: a lista (N3) e a grade
  sabem que uma conversa pertence a um loop sem ler outro fluxo.
- Recusas: workspace inválido → `workspace_invalid`; argumento sem palavra ou
  flag desconhecida → `invalid_input` com a mensagem que a TUI já dá; um loop
  vivo no mesmo workspace → `loop_already_running`, código novo (Q7);
  `continue` fora de `awaiting_review` → `loop_not_awaiting_review`, código
  novo. O protocolo não tem um `conflict` genérico, e um código que diz o que
  houve é o que o cliente consegue dizer à pessoa.

### 4.2 Um evento por ciclo de conferência

`done.checked`, no log da **sessão**, cada vez que `checkDone` roda os critérios:

```go
DoneChecked struct {
    TurnID      string   `json:"turn_id"`
    Cycle       int      `json:"cycle"`        // 1, 2, … dentro do turno
    Met         []string `json:"met,omitempty"`
    Unmet       []string `json:"unmet,omitempty"`
    Unavailable []string `json:"unavailable,omitempty"`
    Moved       string   `json:"moved"`        // forward | stalled | backward
    Stall       int      `json:"stall"`
    StallLimit  int      `json:"stall_limit"`
    Undone      []string `json:"undone,omitempty"` // o que o laço desfez num ciclo que quebrou
    TouchedProtected []string `json:"touched_protected,omitempty"`
}
```

Nunca entra no contexto do modelo (como `Progress` e `StartedAt`, ADR-03).
Turno que não escreveu nada não confere e não emite — é o que `checkDone` já faz.

### 4.3 A superfície morta (Q1, recomendação C)

- **Apagar** `done.signed`, `SignDoneRequest`, `qualifier.Sign` e os dois
  `DCODE_QUALIFIER_*_SIGN_*` do `.config`. A assinatura é o arquivo revisado e
  o `continue`; um ida-e-volta dentro do turno contradiz o desenho que ganhou.
- **Religar** `done.proposed` como **aviso**: emitido pelo `commitDone` no log
  da sessão qualificadora, depois de medir e escrever, com o `DoneProposal`
  medido — sem `ExpiresAt`, sem bloquear turno nenhum (o turno já terminou).
  Ganha `Path` (o arquivo escrito). `TurnID` e `Round` saem. É o que a Figura 3
  desenha sem ler o `Summary` em prosa.
- Como nunca houve emissor, nenhum cliente dependia da forma antiga; ainda
  assim é contrato removido: MINOR, com changelog em `client-server-protocol` e
  `done-qualifier`.

## 5. Mudanças no núcleo

- **`internal/loop/loopcommand`** ganha o `Runner`: a máquina de estados da §4.1,
  com a mesma sequência e as mesmas regras da TUI — survey que falha não impede
  (o changelog "o laço lê as ferramentas"), pasta sem critério qualifica antes,
  proposta escrita para o laço, a última palavra de cada pasta vence
  (`recordLoopResult`), veredito somado (`loopVerdict`). Testes de tabela com um
  `Transport` falso, sem modelo: a sequência é determinística.
- **`internal/app`/`internal/server`**: as rotas, um registro de loops vivos ao
  lado do `session.Manager`, e o fluxo de lista. O loop morre com o daemon —
  nada sobrevive a quem o criou (D19) —, e **não** morre com o cliente que o
  pediu: quem o conduz é o daemon.
- **`internal/loop/turn.go`**: `checkDone` emite `done.checked`.
- **`commitDone`** emite `done.proposed` (aviso).
- **Specs**: changelog em `loop-command` (a sequência sai da TUI; a RN-3 fica:
  reconhecer é do cliente, conduzir é do daemon), `client-server-protocol`
  (rotas, eventos, campos), `done-qualifier` (a assinatura é o arquivo; o resto
  sai). O `.p` de `loop-command` ganha os estados e as transições como
  invariantes com teste nomeado (specguard).
- **TUI**: `/loop` passa a chamar `POST /v1/loops` e a seguir o loop pelo fluxo
  de lista; `loopQueue`, `loopGoal`, `loopWorked`, `loopResults`,
  `loopQualified` e o despacho no `eventMsg` saem. O texto que ela mostra não
  muda — os testes de `loop_command_test.go` e `loop_verdict_test.go` são a régua.

## 6. Mudanças no desktop

- **`src/state/commands.ts`**: `/loop <args>` vira `{ kind: 'loop', args }`; o
  `KNOWN` ganha `/loop`.
- **Processo principal**: canais com nome no `DcodeApi` — `startLoop`,
  `continueLoop`, `stopLoop`, `openProposal` (só daemon local) — e o fluxo
  `/v1/loops/events` ao lado do de conversas. Cada argumento conferido na
  fronteira (D2).
- **`src/state/loop.ts`** (novo, puro): dobra `LoopRun` e `done.checked` no que a
  janela desenha — fase, pasta n de N, ciclo, luzes, contador sem progresso.
  `pane.ts` passa a preferir as luzes do último `done.checked` às do último
  `turn.completed` enquanto o turno roda.
- **Grade**: a janela de um loop segue `LoopRun.session_id`; a régua do loop no
  topo (Figura 2); a folha "Começar loop" (Figura 1); a revisão (Figura 3).
  Atenção (D30): `awaiting_review` conta como **esperando você**.
- **Crew**: linha `↻` na lista; seção Loop na aba Pronto (Figura 4).
- **Decisões**: D44 (o loop é uma janela que segue a sessão corrente, emenda
  D31), D45 (a revisão da proposta), D46 (parar agora/depois). L11 fecha.
- **Réguas**: um cenário `loop` no `check:daemon`, com o modelo roteirizado
  qualificando, propondo e trabalhando uma pasta. `tests/daemon/**` é caminho
  protegido de `docs/loop/done.toml`: o cenário entra num PR próprio, como
  mudança deliberada da régua. As telas 02 e 03 da régua visual não têm loop e
  devem continuar a 0,00%; se mudarem, é outra coisa.

## 7. Perguntas em aberto

**Q1. A superfície morta: religar ou apagar?**
- A. Apagar tudo (`done.proposed`, `done.signed`, `SignDoneRequest`, `Sign`, as
  variáveis). Menor superfície; a revisão estruturada fica sem dado (só o
  `Summary` em prosa).
- B. Religar como o `.p` original previa: assinatura bloqueante dentro do turno,
  com `SignDoneRequest` editando critérios. Contradiz o desenho que ganhou
  (arquivo revisável, turno qualificador em plan) e reabre RN-6 com prazo.
- C. Religar só `done.proposed`, como aviso pós-escrita com o `DoneProposal`
  medido; apagar o resto.
- **Recomendo C**: o desktop precisa da proposta estruturada para a Figura 3, e
  o resto é a assinatura de um desenho abandonado.

**Q2. Onde mora a sequência?**
- A. No daemon (N4, D24): um loop, dois clientes.
- B. Em cada cliente: o desktop reimplementa o que a TUI faz.
- C. Num pacote compartilhado, rodando em cada cliente.
- **Recomendo A.** B e C são duas cópias da mesma regra, e a desta família já
  derivou uma vez (o fim era silêncio, a proposta chegava cedo).

**Q3. Como a pessoa revisa a proposta no desktop?**
- A. Card estruturado + abrir o arquivo no editor + "Começar o trabalho"; o
  daemon mede de novo antes de começar.
- B. Editar os critérios na própria janela (apagar, apertar, trocar comando).
- C. Só o `Summary` em texto.
- **Recomendo A** agora; B é uma proposta futura, porque editar critério é
  editar a régua, e isso merece desenho próprio.

**Q4. Granularidade do progresso.**
- A. Um `done.checked` por conferência dentro do turno.
- B. Só o `turn.completed` (o que existe).
- C. Contagem por teste (`go test: 11/12`), como `LOOP.md` pedia.
- **Recomendo A.** C foi recusada no D24: exigiria ler a saída do comando.

**Q5. O loop na grade.**
- A. Uma janela por loop, que segue a sessão corrente; as pastas terminadas a um
  clique (passo-a-passo).
- B. Cada pasta é uma janela própria (cada sessão uma conversa, D31 ao pé da
  letra).
- C. Uma janela por loop, sem acesso às sessões de cada pasta.
- **Recomendo A**, como emenda ao D31: o que a pessoa acompanha é o loop; B
  enche a grade de janelas terminadas e empurra o resto para fora (D30).

**Q6. Por onde se entra.**
- A. Só `/loop` no campo.
- B. Só a folha "Começar loop".
- C. Os dois, com a mesma gramática.
- **Recomendo C**: o campo é o caminho de quem já usa a TUI; a folha é onde se
  descobre que pasta tem critério.

**Q7. Dois loops no mesmo workspace.**
- A. Recusar (`loop_already_running`) enquanto um estiver vivo.
- B. Permitir.
- C. Permitir com `owns` disjuntos por loop.
- **Recomendo A.** Dois loops na mesma árvore medem o trabalho um do outro; C é
  a mesma pergunta que a proposta do orquestrador responde, e espera ela.

## 8. Entrega em passos (um tema por PR)

| # | Lado | PR | Depende |
|---|---|---|---|
| 1 | núcleo | `refactor(protocol)!: the signature round trip that never shipped is removed` — `done.signed`, `SignDoneRequest`, `qualifier.Sign`, as variáveis; changelog em `done-qualifier` e `client-server-protocol`; `events.ts` regenerado | — |
| 2 | núcleo | `feat(protocol): a written proposal is announced as done.proposed` | 1 |
| 3 | núcleo | `feat(protocol): each check of the criteria inside a turn is an event` (`done.checked`) | — |
| 4 | núcleo | `feat(loop): the daemon runs /loop` — `Runner`, rotas, `LoopRun`, fluxo, `loop_id`; changelog em `loop-command` | 2 |
| 5 | TUI | `refactor(tui): /loop asks the daemon to run it` | 4 |
| 6 | desktop | `feat(desktop): criteria lights move with each check` (lê `done.checked`) | 3 |
| 7 | desktop | `feat(desktop): a loop is started and stopped from the window` (`/loop`, folha, régua do loop, parar) | 4 |
| 8 | desktop | `feat(desktop): a proposed definition of done is reviewed before the work` | 2, 7 |
| 9 | desktop | `test(desktop): the daemon ruler runs a loop` (régua protegida, PR só dela) | 8 |

Cada PR do núcleo leva a linha no `CHANGELOG.md` (as duas edições); cada PR do
desktop, no `desktop/CHANGELOG.md` (as duas) e o D-número em `DECISIONS-3.md`.
O 1 e o 4 são MINOR pelo `scripts/version.sh`.

## 9. Riscos

- **O loop sobrevive ao cliente, não ao daemon.** Fechar o desktop que subiu o
  daemon (D19) mata o loop; a pergunta de fechar (D19) passa a contar loops
  vivos, não só sessões.
- **`awaiting_review` parado para sempre.** É o desenho (RN-6). A grade põe a
  janela em "esperando você", e a lista mostra; nada aprova sozinho.
- **Dois clientes apertam "Começar o trabalho".** O primeiro vence; o segundo
  recebe `loop_not_awaiting_review` e o estado novo pelo fluxo.
- **O passo 5 muda o comportamento da TUI** sem mudar o texto: se algum teste da
  TUI precisar mudar de expectativa, é regressão, não ajuste.
- **`done.checked` caro demais no log** num turno de 40 ciclos com 20 critérios:
  ~800 nomes. Aceitável; se não for, os nomes saem e ficam só as contagens, com
  a lista no `turn.completed`.
- **Specguard**: `LoopRun` e as rotas novas precisam de usuário no mesmo PR
  (`pkg/client` conta), ou de motivo escrito.
- **Régua visual**: a régua do loop no topo da janela muda a tela só quando há
  loop; se uma referência nova for necessária, ela passa pelo visto do dono
  (D18).
