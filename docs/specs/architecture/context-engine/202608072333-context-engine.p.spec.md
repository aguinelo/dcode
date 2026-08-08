# Planning: Motor de Contexto

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608072333-context-engine.r.spec.md`.

## 1. Nível de estabilidade

**`experimental`.** Contrato interno — `internal/context`. Não é superfície pública; promoção a `stable` só se for exposto em `pkg/`.

## 2. A função pura

O componente inteiro se reduz a isto:

```go
// Package: internal/context

// Assemble é PURA: sem I/O, sem relógio, sem aleatoriedade, sem leitura de
// variável de ambiente. Mesma entrada produz saída byte-a-byte idêntica (RN-7).
func Assemble(s Session) ([]Message, error)
```

Qualquer efeito colateral dentro de `Assemble` é achado de arquitetura, não de estilo — ver `docs/conventions/GO-CODE-REVIEW.pt-BR.md`.

## 3. Tipos

```go
type Role string

const (
    RoleSystem    Role = "system"
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
)

type Message struct {
    Role      Role       `json:"role"`
    Text      string     `json:"text,omitempty"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"` // só em RoleAssistant
    ToolResult *ToolResult `json:"tool_result,omitempty"` // só em RoleTool
}

type ToolCall struct {
    ID    string          `json:"id"`
    Name  string          `json:"name"`
    Input json.RawMessage `json:"input"`
}

type ToolResult struct {
    ToolCallID string `json:"tool_call_id"`
    Output     string `json:"output"`
    IsError    bool   `json:"is_error"`
    Truncated  bool   `json:"truncated"`
}

// Session é o estado completo do qual o contexto é função.
type Session struct {
    Instructions string      // system prompt, fixado na criação
    Tools        []ToolDef   // fixado na criação (RN-3)
    Summary      *Summary    // resultado da última compactação; nil se nunca houve
    History      []Message   // append-only (RN-1)
}

type ToolDef struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Schema      json.RawMessage `json:"schema"`
}

type Summary struct {
    Text     string // resumo do trecho compactado
    UpToIdx  int    // índice exclusivo em History coberto pelo resumo
}
```

## 4. Ordem de montagem

Fixa, do mais estável para o mais volátil (RN-4). É o que maximiza o prefixo casável com cache.

| # | Seção | Origem | Muda quando |
|---|---|---|---|
| 1 | Instruções de sistema | `Session.Instructions` | nunca, dentro da sessão |
| 2 | Definições de ferramenta | `Session.Tools` | nunca, dentro da sessão (RN-3) |
| 3 | Resumo de compactação | `Session.Summary` | só na compactação (RN-5) |
| 4 | Histórico vivo | `History[Summary.UpToIdx:]` | a cada turno, só por append |

Quando `Summary` é `nil`, a seção 3 é omitida por completo — não emite marcador vazio, que seria diferença de bytes.

**Proibido no prefixo (RN-2):** timestamp, contagem de token, número de iteração, ID de sessão, estado de conexão, caminho absoluto que varie por máquina.

## 5. Compactação

```go
// Plan decide SE compactar e ONDE cortar. Pura.
func Plan(s Session, cfg Config) (CompactionPlan, bool)

type CompactionPlan struct {
    FromIdx int // índice inclusivo do início do trecho a compactar
    ToIdx   int // índice exclusivo do fim
    Keep    []int // índices dentro do trecho preservados na íntegra (RN-6)
}
```

**Gatilho:** tokens estimados do contexto montado ≥ `CompactAt` × janela do modelo.

**Ponto de corte:** `ToIdx` cai sempre em fronteira de turno completo — nunca separa um `RoleAssistant` com `ToolCalls` dos seus `RoleTool` correspondentes. Um turno partido produz histórico inválido para o provedor.

**Preservação obrigatória (RN-6):** a mensagem `RoleUser` mais recente e todas as posteriores a ela nunca entram no trecho compactado. A tarefa corrente sobrevive por construção, não por qualidade do resumo.

**Aplicação:** `Plan` é pura e decide. A geração do texto do resumo chama o modelo, portanto **não** pertence a este componente — o loop do agente executa o plano e devolve `Summary` preenchido. Essa separação é o que mantém `Assemble` e `Plan` puras.

## 6. Estimativa de tokens

```go
// Estimate é aproximada e determinística. Não chama tokenizador de rede.
func Estimate(msgs []Message) int
```

Aproximação por heurística de caracteres, com margem conservadora. Precisão exata não é necessária: o gatilho é uma fração da janela e a margem absorve o erro. O que **é** necessário é determinismo — a mesma entrada sempre estima o mesmo valor, senão o golden test da compactação fica instável.

## 7. Invariantes verificáveis

Cada linha é caso de teste obrigatório.

- `Assemble` chamada duas vezes com a mesma `Session` produz slices byte-a-byte idênticos.
- Anexar uma mensagem a `History` não altera nenhum byte do prefixo previamente produzido (RN-1). Verificado comparando o prefixo comum antes e depois.
- Nenhuma saída de `Assemble` contém dígito de timestamp, contador ou ID de sessão (RN-2).
- Alterar `Session.Tools` após a criação é erro; a sessão não expõe caminho para isso (RN-3).
- A ordem das seções é sempre a da tabela 4, para qualquer combinação de campos presentes ou ausentes.
- `Summary == nil` produz saída sem qualquer marcador de resumo.
- `Plan` nunca devolve `ToIdx` que separe `RoleAssistant` com `ToolCalls` dos seus `RoleTool`.
- `Plan` nunca inclui a última `RoleUser` nem nada posterior a ela no trecho compactado (RN-6).
- `Estimate` é determinística para a mesma entrada.
- `Assemble` não realiza I/O: verificado por teste que falha se o pacote importar `os`, `net` ou `time` fora de tipos.

## 8. Changelog

_Sem alterações desde a criação._
