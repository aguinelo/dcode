# Orçamento de contexto realimentado

**Data:** 2026-08-10
**Specs afetadas:** `202608072333-context-engine` (`.r`, `.p`, `.config`), `202608080016-behavior-definition` (`.p`)

> **Regra:** o agente sabe quanto do orçamento já gastou **antes** de acabar, não depois.

## O problema medido

O harness sabe a fração exata da janela. `internal/contextengine/compaction.go:11` calcula o total e a linha 54 o compara com o gatilho:

```go
if float64(Estimate(msgs, cfg)) < cfg.CompactAt*float64(cfg.Window) {
```

Essa conta roda **a cada iteração do loop**. O número existe, está sempre atualizado, e **nunca chega ao modelo**.

O único sinal que o modelo recebe está em `internal/behavior/reminders.go:85`, e o nome diz tudo:

```go
Kind: ReminderCompacted,
```

**No passado.** Ele descobre que a memória foi cortada *depois* do corte — como o aluno que descobre que a folha acabou ao ser interrompido no meio da frase. Nunca a tempo de fazer algo a respeito.

É o terceiro caso do mesmo padrão no produto, e agora o formato é visível:

| Onde | O harness sabe | O modelo recebe |
|---|---|---|
| verificação (`202608102000`) | se rodou e o código de saída | nada |
| diff de edição | o diff exato, já calculado | `+3 −1` |
| **janela de contexto** | **a fração exata** | **só o aviso pós-corte** |

Três vezes a mesma forma: **o produto tem o fato e não o entrega a quem decide.**

## O que muda de comportamento

Um agente que sabe estar em ~75% age diferente, e são ações concretas:

- **anota o que descobriu** em arquivo, antes de a descoberta ser resumida embora
- **fecha o que está aberto** em vez de abrir mais uma frente
- **avisa o usuário** que a tarefa é maior que o espaço restante — antes, não depois
- **para de ler arquivo inteiro** e passa a ler o trecho

Nenhuma dessas é possível quando o único sinal chega após o corte.

## As três restrições que decidem o desenho

### 1. Não pode ir para o prefixo

A RN-1 e a RN-2 desta spec proíbem: o prefixo é imutável e nada volátil entra nele. Um número que muda a cada turno **invalidaria o cache em todo turno** — pagar-se-ia o prompt inteiro sempre, para economizar contexto. Trocar um problema por outro maior.

Vai pelo **canal de lembrete** (RN-6 de `behavior-definition`), anexado ao histórico. É exatamente o problema para o qual aquele canal existe.

### 2. Faixa, não valor exato

Texto de lembrete é **constante por `Kind`**, com interpolação apenas de dado já presente no histórico — caminho de arquivo sim, contador não. Valor exato mudando a cada turno gera texto sempre novo e o histórico deixa de ser reproduzível, quebrando a RN-7 desta spec.

Faixas fixas, e o texto se repete enquanto nada muda de verdade:

| Faixa | Quando cruza | O que o lembrete pede |
|---|---|---|
| `60%` | primeira travessia | preferir trecho a arquivo inteiro |
| `80%` | primeira travessia | registrar em arquivo o que precisa sobreviver; fechar o que está aberto |
| `92%` | primeira travessia | dizer ao usuário que a tarefa não cabe no que resta |

O último limiar fica **abaixo** de `CompactAt`, senão o aviso chega junto com o corte e não serve para nada.

### 3. Disparo por borda, nunca por nível

Emitir enquanto a fração estiver acima do limiar repetiria o mesmo lembrete em todo turno — custo crescente e, pior, **habituação**: um aviso que aparece sempre deixa de ser lido.

Emite-se **uma vez, na travessia para cima**. A faixa mais alta já emitida vive no estado da sessão, o que mantém `Emit` puro (RN-6): mesmo estado, mesmo conjunto.

**Rearma na compactação.** Depois do corte a fração despenca; voltar a subir é informação genuinamente nova, e o segundo aviso é tão útil quanto o primeiro.

## Tipos

```go
// internal/contextengine

// Fraction é quanto da janela o contexto montado ocupa, em [0,1].
// Determinística, como Estimate — é a mesma conta que Plan já faz.
func Fraction(s Session, cfg Config) float64

// Band é a faixa de ocupação já anunciada ao modelo. Vive no estado da sessão
// para que a emissão seja por borda e Emit continue puro.
type Band int

const (
    BandNone Band = iota // nada anunciado
    Band60
    Band80
    Band92
)

// BandFor devolve a faixa correspondente a uma fração. Pura.
func BandFor(f float64) Band
```

```go
// internal/behavior — junto dos demais Kind
const ReminderContextBudget ReminderKind = "context_budget"
```

Um `Kind` só, com três textos selecionados pela `Band` — não três `Kind`. A regra é uma; o que muda é o quanto resta.

## Fronteira de determinismo

| Parte | Regime | Verificação |
|---|---|---|
| `Fraction` e `BandFor` | determinístico | asserção |
| emissão por borda a partir do estado | determinístico | asserção |
| rearme na compactação | determinístico | asserção |
| ausência do número no prefixo | determinístico | varredura |
| **agir sobre o aviso** | **mediado** | limiar |

## Contratos comportamentais

| ID | Cenário | Comportamento esperado | Limiar |
|---|---|---|---|
| `records-before-compaction` | lembrete de `80%` durante tarefa longa | registra em arquivo o que precisa sobreviver ao resumo | ≥ 85% |
| `warns-when-task-exceeds-budget` | lembrete de `92%` com tarefa claramente maior | diz ao usuário que não cabe, em vez de continuar | ≥ 90% |
| `no-budget-noise-when-low` | sessão curta, bem abaixo de `60%` | **nenhum** lembrete de orçamento | **100%** |

`no-budget-noise-when-low` a 100% porque é determinístico: nada abaixo do primeiro limiar emite, e isso é asserção, não limiar de modelo.

## Impacto

- `Fraction`, `Band` e `BandFor` em `internal/contextengine`; nenhuma mudança em `Estimate` nem em `Plan`.
- Faixa anunciada passa a fazer parte do estado da sessão, do lado do servidor.
- Um `ReminderKind` novo; nada sai do prefixo e nada entra nele.
- `ReminderCompacted` **permanece** — dizer que o corte aconteceu continua sendo necessário. O que muda é deixar de ser o único sinal.
- O cliente já exibe ocupação; nenhuma mudança de interface é exigida por esta.
