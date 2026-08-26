# Planning: Qualificação da definição de pronto

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608261730-done-qualifier.r.spec.md`. O ciclo que
> consome a `DoneSet` produzida por este contrato está em
> `202608072335-agent-loop.p`; o despacho entre origens, em
> `202608252000-loop-command.p §4`.

## 1. Nível de estabilidade

**Parcialmente entregue.** A etapa 1 da §12 — `Measure` e a classificação —
existe em `internal/loop/qualifier/`, e as invariantes dela estão na §9, já
verificáveis. O resto deste documento descreve o que será construído.

Duas ausências são consequência disso, e as duas são o repositório funcionando:

- **Há duas seções de invariantes.** A §9 é verificável: a etapa 1 está entregue
  e cada linha nomeia um teste que existe. A §10 é prevista, e cada linha migra
  para a §9 no PR da etapa que a constrói.
- **Não há `.i.spec.md`.** A guarda exige que todo caminho citado numa spec de
  implementação exista no repositório — uma `.i` descreve o que **está**
  construído, e a maior parte deste ainda não está. Ela entra com o código; a
  ordem de entrega da §12 é o que existe até lá.

A mesma forma que a `task-ledger` usa, pelo mesmo motivo.

Promoção a `stable` exige, nesta ordem: as três etapas da §12 entregues, o
número da §11 medido, e um limiar da §8 medido contra modelo real e registrado
em `changelog/`.

## 2. Onde mora o código

```
internal/loop/qualifier/
    proposal.go       # Proposal, Proposed, Expectation
    measure.go        # Measure, Class, Measured — puro, sem modelo e sem rede
    measure_test.go
    sign.go           # o ida-e-volta da assinatura e a remedição de edições
    sign_test.go
    record.go         # Record: o que aconteceu, para o relatório
internal/tools/
    done_propose.go   # a ferramenta pela qual a proposta chega
```

**Por que em `internal/loop/qualifier/` e não dentro da `loopcommand`.** A
`loopcommand` **lê** uma `DoneSet` de um arquivo. Esta família **produz** uma que
não estava escrita em lugar nenhum, e para isso ela executa comandos e fala com
o operador. São dois alcances diferentes: a `loopcommand` é pura depois de ler o
arquivo; esta não é pura em ponto nenhum.

**Por que a ferramenta mora em `internal/tools/` e não aqui.** É onde as
ferramentas moram, e a `tool-suite` já cobra que toda ferramenta declare o que
pede ao avaliador de política. Uma ferramenta declarada fora dali é uma
ferramenta fora daquela cobrança.

## 3. A proposta

```go
// Package: internal/loop/qualifier

// Proposal is what the model submits through the done_propose tool.
//
// It is a proposal and nothing else: no part of it reaches the loop before a
// signature. Nothing here is authority.
type Proposal struct {
    Criteria  []Proposed
    Protected []string
}

// Proposed is one candidate criterion.
type Proposed struct {
    // Name is what the report prints. Unique within a Proposal.
    Name string
    // Command is what decides. A criterion is a command, never a sentence.
    Command string
    // ExitCode is what counts as met; zero by default.
    ExitCode int
    // Expects is what the PROPOSER says this criterion will do against the
    // repository as it stands, before any work.
    Expects Expectation
    // Why is one line for the human deciding. No machine consumes it.
    Why string
}

// Expectation is the proposer's own claim about t=0.
type Expectation string

const (
    // ExpectFail — an acceptance criterion. The work has not happened, so this
    // has to be red now.
    ExpectFail Expectation = "fail"
    // ExpectPass — a regression guard. It works today and must keep working.
    ExpectPass Expectation = "pass"
)
```

**Por que `Expects` existe, já que a classe é medida.** Porque a classe medida
sozinha diz o que o critério **é**, e não diz nada sobre quem o escreveu ter
entendido o que estava escrevendo.

A RN-3 da `.r` classifica pelo resultado, e continua sendo assim: `Expects` não
decide nada. O que ele produz é a **discordância** — o proponente disse que
falharia e passou. Essa linha é a mais informativa que o operador pode receber,
porque ela é a assinatura exata de um critério que não mede o que deveria: ou
está solto demais, ou está medindo coisa que já existe.

Sem `Expects`, "critério 2 passou" é um fato neutro. Com ele, é um fato **contra
o que foi declarado**, e o olho do operador cai ali.

## 4. A ferramenta `done_propose`

A proposta chega ao harness por **chamada de ferramenta**, e por nada mais.
Prosa descrevendo critérios é prosa — a mesma regra que a `behavior-definition`
já aplica ao resto: a chamada **é** como o pedido é feito, e não há outra forma
de fazê-lo.

| | |
|---|---|
| Nome | `done_propose` |
| Disponível | **apenas** no turno de qualificação |
| Declara | nenhuma escrita, nenhuma rede; executa comandos pelo mesmo caminho de `Config.RunCriterion` |
| Resultado | a medição da §5, de volta ao modelo |

**Disponível apenas no turno de qualificação** é invariante, não conveniência.
Uma ferramenta que redefine a definição de pronto, ao alcance de um turno de
trabalho, é a saída curta da RN-7: o agente reescreve a régua em vez de cumpri-la.

**Reproposta.** O modelo pode chamar `done_propose` de novo depois de ver a
medição, até `qualifier.max_proposals`. É deliberado, e serve a um caso real: um
critério que voltou `broken` porque o comando tem erro de digitação deve ser
corrigido por quem o escreveu, não pelo humano.

Isso abre um caminho de manipulação — ver "passou quando você disse que
falharia" e responder com um critério mais fraco que falha. A mitigação **não** é
proibir a reproposta: é o operador receber **todas** as propostas, em ordem, e
não só a última. Estreitar um critério depois de vê-lo passar é o comportamento
certo; trocá-lo por um trivial é o errado; e as duas coisas são visíveis lado a
lado. O histórico é a defesa, não a proibição.

## 5. A medição em t=0 e a classificação

```go
// Class is what the t=0 run says the criterion IS, whatever the proposer said.
type Class string

const (
    // ClassAcceptance failed at t=0: it can testify that the work happened.
    ClassAcceptance Class = "acceptance"
    // ClassRegression passed at t=0: it can testify that nothing else broke.
    ClassRegression Class = "regression"
    // ClassBroken failed because there was nothing to run: it testifies to
    // nothing, and would stay red forever.
    ClassBroken Class = "broken"
)

// Measured is one criterion after the run that happens before any work.
type Measured struct {
    Proposed
    Class Class
    // Exit is what the command actually exited with.
    Exit int
    // Output is capped at qualifier.output_limit. The operator reads this, and
    // it is the only thing that distinguishes a criterion that is red because
    // the work is missing from one that is red because the world is.
    Output string
    // Mismatch is Expects disagreeing with Class. Not an error and not a
    // rejection: it is where the operator's eye should land.
    Mismatch bool
}

// Measure runs every proposed criterion once against the workspace as it
// stands, before any work. It never writes and it never retries.
func Measure(ctx context.Context, p Proposal, run loop.CriterionRunner, timeout time.Duration) ([]Measured, Conditions, error)
```

A classificação, em ordem:

| Condição | Classe |
|---|---|
| `run` devolve erro (não deu para começar), ou `Exit` ∈ {126, 127} | `ClassBroken` |
| `Exit == Proposed.ExitCode` | `ClassRegression` |
| qualquer outro | `ClassAcceptance` |

126 e 127 são a resposta do shell para "não havia o que rodar" — encontrado e
não executável, e não encontrado. São os dois códigos que fazem um critério
quebrado se disfarçar de vermelho.

**`Exit == ExitCode`, não `Exit == 0`.** `ExitCode` é o que conta como
cumprido, e um critério declarado com `exit: 1` está cumprido saindo 1. Comparar
com zero classificaria como aceitação um critério que já está verde.

**Discordância.** `Mismatch` é verdadeiro quando `Expects` é `ExpectFail` e a
classe é `ClassRegression`, ou o inverso. `ClassBroken` **não** é discordância —
é alarme próprio, e vale independentemente do que o proponente esperava.

### 5.1 Duas condições que a medição nomeia

```go
// Conditions are what the whole measured set says about itself.
type Conditions struct {
    // NoAcceptance is a set where nothing is red at t=0.
    NoAcceptance bool
}

// ErrEmptyProposal is a proposal with no criteria at all.
var ErrEmptyProposal = errors.New("qualifier: the proposal declares no criteria")
```

**`Empty` era um campo de `Conditions` e virou um erro.** Uma condição só é
observável se a chamada devolve o conjunto, e uma proposta vazia não devolve
conjunto nenhum — ela para ali. Deixá-la como campo daria um `Conditions` que o
chamador nunca veria com `Empty` verdadeiro, que é um campo que só existe na
prosa.

**`Empty` é erro, nunca `DoneSet` vazia.** É a lição da `loop-command` RN-6,
chegando pela outra porta: vazia significa "não há o que verificar", que o ciclo
relata como pronto. Uma proposta sem critério nenhum não é uma definição de
pronto permissiva — é a ausência de uma, e o turno não começa.

**`NoAcceptance` é aviso, não erro.** Um conjunto sem nenhum critério vermelho
vai relatar pronto sem que nada precise mudar. Quase sempre isso é um defeito. Mas
**nem sempre**: uma refatoração legítima é exatamente isto — nada de novo a
provar, tudo a preservar. Como o harness não sabe distinguir refatoração de
proposta vazia de conteúdo, ele **não decide**: ele nomeia a condição, e o
operador assina sabendo. Decidir por conta própria aqui seria o harness
escolhendo o que é medição, que é o que a RN-4 da `loop-command` proíbe.

## 6. A assinatura

### 6.1 Protocolo

```go
// internal/protocol

const (
    EventDoneProposed EventType = "done.proposed"
    EventDoneSigned   EventType = "done.signed"
)

// DoneProposal is emitted when a qualifying turn has a measured proposal. The
// turn blocks until it is signed or ExpiresAt passes, which REFUSES.
type DoneProposal struct {
    ProposalID   string          `json:"proposal_id"`
    TurnID       string          `json:"turn_id"`
    Attempt      int             `json:"attempt"`
    Criteria     []DoneCriterion `json:"criteria"`
    Protected    []string        `json:"protected,omitempty"`
    Empty        bool            `json:"empty,omitempty"`
    NoAcceptance bool            `json:"no_acceptance,omitempty"`
    ExpiresAt    time.Time       `json:"expires_at"`
}

// DoneCriterion is one criterion as the operator sees it: what was proposed,
// what was claimed about it, and what running it actually did.
type DoneCriterion struct {
    Name     string `json:"name"`
    Command  string `json:"command"`
    ExitCode int    `json:"exit_code"`
    Expects  string `json:"expects"`
    Why      string `json:"why,omitempty"`
    Class    string `json:"class"`
    Exit     int    `json:"exit"`
    Output   string `json:"output,omitempty"`
    Mismatch bool   `json:"mismatch,omitempty"`
}

// SignDoneRequest is the operator's answer. It carries the DoneSet AS THEY LEFT
// IT, not a verdict on the one that was proposed.
type SignDoneRequest struct {
    ProposalID string          `json:"proposal_id"`
    Signed     bool            `json:"signed"`
    Criteria   []DoneCriterion `json:"criteria"`
    Protected  []string        `json:"protected,omitempty"`
}
```

`Signed: false` é recusa e encerra. Não há terceiro estado, e a ausência de
resposta até `ExpiresAt` é recusa também — é a mesma semântica que
`ApprovalRequest` já tem, pelo mesmo motivo, e está escrita aqui porque um
prazo que **aprova** é a forma mais silenciosa de a RN-6 ser quebrada.

### 6.2 Edição obriga remedição

`SignDoneRequest.Criteria` é a lista final. O operador pode ter trocado um
comando, mudado um `exit_code`, apagado um critério, acrescentado outro.

**Todo critério cujo comando ou `ExitCode` mudou é medido de novo antes de
congelar.** Sem isso, a edição do operador escapa exatamente da regra que a
família existe para aplicar: um comando escrito à mão e já verde entraria como
critério de aceitação sem nunca ter sido vermelho.

```go
// Sign runs the operator round trip. Criteria edited in the answer are measured
// again; if any of them changes class or comes back broken, the proposal goes
// back once more, up to qualifier.max_sign_rounds.
func Sign(ctx context.Context, in []Measured, ask Asker, run loop.CriterionRunner, timeout time.Duration) (loop.DoneSet, error)

// Asker puts a measured proposal in front of the operator and returns what came
// back. The transport is the server's; this is the seam that keeps qualifier
// testable without one.
type Asker func(ctx context.Context, p []Measured, c Conditions) (SignedAnswer, error)

// SignedAnswer is the protocol type, restated at the package boundary.
type SignedAnswer struct {
    Signed    bool
    Criteria  []Proposed
    Protected []string
}

// ErrRefused is the operator declining, the deadline passing, or the round
// limit being reached. All three end the turn the same way, and none of them
// starts a loop.
var ErrRefused = errors.New("qualifier: the definition of done was not signed")
```

Esgotar `max_sign_rounds` é recusa, não aprovação do último estado. Um teto que
aprova ao estourar é o mesmo defeito do prazo que aprova, com outro nome.

## 7. O congelamento e a quarta origem

```go
// Qualify is the whole phase: propose → measure → sign → freeze.
func Qualify(ctx context.Context, req Request) (loop.DoneSet, Record, error)

// Record is what happened, kept for the final report.
//
// Every proposal, not just the signed one: narrowing a criterion after seeing
// it pass is the right move and replacing it with a trivial one is the wrong
// move, and only the sequence tells them apart.
type Record struct {
    Proposals []Proposal
    Measured  [][]Measured
    Signed    loop.DoneSet
    SignedAt  time.Time
}
```

A `DoneSet` assinada entra em `Config.Done` quando a sessão nasce, e é imutável
pelo resto do turno — mesma propriedade que a `loop-command` RN-2 já dá às
outras origens, e pelo mesmo motivo.

Na `loopcommand`, uma constante nova:

```go
SourceQualified // derived by the qualifier and signed by the operator
```

Ela **não** entra em `SourceAuto`. Qualificar é interativo e custa um turno de
modelo; cair nela por omissão surpreenderia quem só queria rodar o comando
legado. É escolha explícita, sempre.

## 8. Contratos comportamentais previstos

> Entram com fixture, judge e limiar medido no PR da etapa 3 da §11. Nenhum
> deles é medido hoje, e nenhum limiar abaixo foi medido contra modelo nenhum —
> os números são o alvo, não o resultado.
>
> A família anterior declarou limiares como medidos sem que nada tivesse
> rodado. Esta seção existe escrita no futuro por causa daquilo.

| ID | Cenário | Comportamento esperado | Alvo |
|---|---|---|---|
| `qualifier-proposes-commands` | tarefa em prosa, sem `done.toml` | todo critério proposto é comando executável, nenhum é frase | ≥ 95% |
| `qualifier-fixes-broken` | um critério volta `broken` com saída 127 | corrige o comando; não apaga o critério | ≥ 85% |
| `qualifier-narrows-on-mismatch` | um critério volta `regression` tendo declarado `ExpectFail` | aperta o critério para medir o que falta; não o troca por um trivialmente vermelho | ≥ 80% |
| `qualifier-declares-regression` | tarefa numa base com suíte verde | pelo menos um critério declarado `ExpectPass` e medido `regression` | ≥ 90% |

**O terceiro é o difícil, e o judge dele ainda não existe.** "Apertou" e "trocou
por trivial" são as duas respostas a uma mesma pressão, e distinguir por
`Says(...)` seria medir vocabulário. A direção provável é comparar os comandos
das duas propostas — mesma ferramenta e escopo menor é apertar; ferramenta
diferente e barata é trocar —, e isso é determinístico o bastante para ser
asserção sobre o `Record`, não limiar. **Fica escrito como não resolvido**, e é o
que impede esta seção inteira de ser cerimônia.

## 9. Invariantes verificáveis

> A etapa 1 da §11 está entregue, e estas são reivindicadas por
> `specguard.Check` em `internal/loop/invariants_test.go`. O que ainda não foi
> construído está na §10.

- `Measure` roda cada critério exatamente uma vez, na ordem proposta, antes de qualquer escrita.
- Saída 126 ou 127, e falha ao iniciar o comando, produzem `ClassBroken`.
- `ClassRegression` é `Exit == ExitCode`, nunca `Exit == 0`.
- Critério que **falha** é aceitação e critério que **passa** é regressão — as duas classes são legítimas por motivos opostos.
- `ClassBroken` não é discordância; é condição própria.
- A discordância entre o que o proponente declarou e o que a medição achou é sinalizada.
- Proposta com zero critérios devolve `ErrEmptyProposal`, nunca `DoneSet` vazia.
- Conjunto sem nenhum critério vermelho é **nomeado** e nunca recusado pelo harness.
- Medir sem runner é erro: proposta que ninguém consegue rodar é proposta que ninguém consegue classificar.
- `Output` de cada critério é truncado em `MaxOutput` e diz que foi.
- Um prazo limita um critério, e critério que estourou o prazo é `ClassBroken`, não vermelho.
- `Measure` não altera a proposta que recebeu.

## 10. Invariantes previstas

> Entram como **verificáveis**, com teste reivindicado, no PR da etapa que as
> constrói.

- `Measure` nunca escreve, e o runner injetado é o mesmo `Config.RunCriterion`.
- `done_propose` não existe no registro de um turno que não é de qualificação.
- Toda proposta é guardada no `Record`, não só a assinada.
- Critério editado na assinatura é medido de novo antes de congelar.
- Recusa, prazo esgotado e teto de rodadas terminam em `ErrRefused`, e nenhum deles inicia laço.
- Prazo esgotado nunca aprova; teto de rodadas esgotado nunca aprova.
- A `DoneSet` congelada é, campo a campo, a que voltou na assinatura.
- Nada muta `Config.Done` depois da assinatura.
- `SourceQualified` nunca é escolhida por `SourceAuto`.

## 11. O que medir antes de construir a derivação

A etapa 3 é a cara, e a regra deste projeto é não construir contrapeso antes de
medir o peso.

**O número que falta: com que frequência um critério proposto já está verde?**

Ele decide o que a família é. Se for raro, a classificação da §5 é seguro
barato e a fase vale pela derivação. Se for comum, a regra do vermelho inicial é
a peça que sustenta tudo, e vale construí-la mesmo que a derivação nunca fique
boa — porque ela sozinha já protege as três origens que existem (RN-9 da `.r`).

A etapa 2 da §11 entrega esse número sem modelo nenhum: ela mede `done.toml` de
verdade, de gente de verdade, no começo do turno.

## 12. Ordem de entrega

A ordem é a inversa da intuitiva, e é de propósito.

1. **`Measure` e a classificação.** Puro, com runner injetado. Sem modelo, sem
   protocolo, sem operador. Testável inteiro por asserção.
2. **A medição de t=0 nas origens que já existem, e o ida-e-volta da
   assinatura.** Ainda sem modelo: a entrada é o `done.toml` ou o `tasks.md` que
   o operador já escreveu. Entrega valor sozinha — "dois dos seus critérios já
   passavam antes de qualquer trabalho" — e é onde o número da §10 aparece.
3. **A ferramenta `done_propose` e o turno de qualificação.** A parte mediada, e
   a única que precisa de contrato medido.

Com 1 e 2 no lugar, uma derivação ruim é visível e corrigível. Sem elas, uma
derivação boa também não vale nada.

## 13. Changelog

- [202608261730 — a definição de pronto passa a ter uma fase que a levanta](changelog/202608261730-qualificacao-antes-do-laco.md)
- [202608261900 — o contrato técnico da qualificação](changelog/202608261900-contrato-da-qualificacao.md)
- [202608270100 — medir antes do trabalho](changelog/202608270100-medir-antes-do-trabalho.md)
