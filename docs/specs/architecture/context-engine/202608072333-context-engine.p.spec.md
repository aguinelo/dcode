# Planning: Motor de Contexto

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608072333-context-engine.r.spec.md`.

## 1. Nível de estabilidade

**`experimental`.** Contrato interno — `internal/contextengine` (o nome evita sombrear o pacote `context` da biblioteca padrão, que todo arquivo do projeto importa). Não é superfície pública; promoção a `stable` só se for exposto em `pkg/`.

## 2. A função pura

O componente inteiro se reduz a isto:

```go
// Package: internal/contextengine

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
    Role       Role        `json:"role"`
    Text       string      `json:"text,omitempty"`
    ToolCalls  []ToolCall  `json:"tool_calls,omitempty"` // só em RoleAssistant
    ToolResult *ToolResult `json:"tool_result,omitempty"` // só em RoleTool
    Images     []Image     // imagens lidas com a mensagem, em bytes (RN-7)
    Documents  []Document  // só PDFs; texto já entrou em Text (ver 3.1)
    // Reminder marca mensagem que o harness anexou, e não que a pessoa
    // digitou. Viaja no papel user, o único que todo provedor aceita no meio
    // da conversa, e nenhum cliente pode exibi-la como fala da pessoa.
    Reminder   bool        `json:"reminder,omitempty"`
}

// Image é carregada em bytes, nunca por caminho: caminho varia por máquina e
// pode mudar sob a sessão, e Assemble tem de ser byte-a-byte reprodutível.
type Image struct {
    MediaType string // image/png, image/jpeg, image/gif, image/webp
    Data      []byte
}

// Document é arquivo lido com um turno, em bytes pelo mesmo motivo.
type Document struct {
    Name      string // como a pessoa o anexou; mostrado ao modelo ao lado
    MediaType string
    Data      []byte
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
    Text    string `json:"text"`      // resumo do trecho compactado
    UpToIdx int    `json:"up_to_idx"` // índice exclusivo em History coberto pelo resumo
}

// Config carrega o que Plan, Estimate e Fraction precisam. Passada, e não
// lida do ambiente, para que as funções continuem puras. Valores fixos em
// DefaultConfig — ver `.config`, seção 3.
type Config struct {
    CompactAt     float64 // fração da janela que dispara a compactação
    KeepTurns     int     // turnos recentes preservados além dos obrigatórios
    KeepFraction  float64 // fração da janela que a cauda preserva, em tokens
    CharsPerToken float64 // heurística da estimativa
    Margin        float64 // somada a toda estimativa, absorve o erro
    Window        int     // janela do modelo em tokens; vem do provider
    Disabled      bool    // desliga só o gatilho automático (zero = ligado)
}

func DefaultConfig() Config
```

### 3.1 Documentos anexados

```go
// InlineDocuments escreve os arquivos de texto na pergunta e devolve os PDFs,
// que a família precisa declarar que lê. PURA.
func InlineDocuments(text string, docs []Document) (string, []Document)
```

Todo `Document` cujo `MediaType` começa com `text/` entra **antes** da pergunta, cada um entre marcadores, nesta forma exata:

```
<document name="NOME" media_type="TIPO" boundary="B">
…bytes do arquivo, intactos…
</document boundary="B">

```

- `NOME` e `TIPO` passam por `html.EscapeString`: aspas no nome são nome, não começo de atributo.
- `B` são os primeiros 12 bytes do SHA-256 do conteúdo, em hexadecimal (24 caracteres). Derivado do conteúdo, e não aleatório, para que o mesmo arquivo dê a mesma mensagem; um corpo não consegue conter o marcador de fechamento sem conter o próprio digest, então nada nele encerra o bloco antes da hora — um `</document>` dentro do arquivo continua dentro do arquivo.
- Arquivo que não termina em `\n` ganha um antes do fechamento; cada bloco termina com uma linha em branco.

O que não é `text/` — na prática, PDF — volta em `Documents` e viaja ao lado da mensagem.

```go
// PDFPages estima as páginas de um PDF pelos objetos de página. Heurística:
// PDF que comprime os objetos esconde as páginas e conta como uma, nunca zero.
func PDFPages(data []byte) int
```

Na estimativa, cada PDF custa `len(Name) + PDFPages(Data) × charsPerPDFPage` caracteres (`.config`, seção 3).

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
}

// Apply devolve a sessão com o plano aplicado e text como novo resumo, somado
// ao anterior. Não altera o slice History de quem chama.
func Apply(s Session, plan CompactionPlan, text string) Session
```

**Gatilho:** tokens estimados do contexto montado ≥ `CompactAt` × janela do modelo.

**Ponto de corte:** `ToIdx` cai sempre em fronteira de turno completo — nunca separa um `RoleAssistant` com `ToolCalls` dos seus `RoleTool` correspondentes. Um turno partido produz histórico inválido para o provedor.

**Preservação obrigatória (RN-6):** a mensagem `RoleUser` mais recente e todas as posteriores a ela nunca entram no trecho compactado. A tarefa corrente sobrevive por construção, não por qualidade do resumo.

**Aplicação:** `Plan` é pura e decide. A geração do texto do resumo chama o modelo, portanto **não** pertence a este componente — o loop do agente pede o resumo e, **só com um resumo inteiro**, chama `Apply`. Essa separação é o que mantém `Assemble` e `Plan` puras.

**Resumo que falha não compacta.** Resumo que volta com erro, vazio ou cortado no teto de saída não aplica o plano: o histórico fica inteiro, e o loop diz por quê em `session.error` `compaction_failed`. A decisão e o que cada caminho faz depois — o gatilho automático segue o turno, `/compact` devolve o erro, a recusa `context_size` encerra — estão em `agent-loop/changelog/202610161200-resumo-que-falha-nao-compacta.md`. Perder detalhe por resumo ruim nunca é trocado por perder o trecho em silêncio.

## 6. Estimativa de tokens

```go
// Estimate é aproximada e determinística. Não chama tokenizador de rede.
func Estimate(msgs []Message, cfg Config) int
```

```go
// Fraction é quanto da janela o contexto montado ocupa, em [0,1]. É a mesma
// conta que Plan já faz, exposta (RN-6.1).
func Fraction(s Session, cfg Config) float64

// Band é a faixa já anunciada ao modelo. Vive no estado da sessão para que a
// emissão seja por BORDA e o emissor de lembrete continue puro.
type Band int

const (
    BandNone Band = iota
    Band60
    Band80
    Band92
)

// BandFor é pura.
func BandFor(f, compactAt float64) Band

// Crossed devolve a faixa a anunciar e se anuncia. Só para cima; descer — o
// que a compactação faz — não anuncia, mas rearma.
func Crossed(announced Band, f, compactAt float64) (Band, bool)
```

> **Correção de assinatura.** `Estimate` recebe `Config` porque a heurística — caracteres por token e margem — é configuração, e lê-la de um global tornaria a função impura e a compactação não reproduzível.
>
> `BandFor` recebe `compactAt` porque as faixas são fração do **orçamento**, não da janela, e o orçamento acaba onde a compactação começa. Ver a nota no `.config`: presas à janela, duas das três seriam inalcançáveis.


O limiar mais alto é **menor** que `CompactAt`, senão o aviso chega junto com o corte.

Aproximação por heurística de caracteres, com margem conservadora. Precisão exata não é necessária: o gatilho é uma fração da janela e a margem absorve o erro. O que **é** necessário é determinismo — a mesma entrada sempre estima o mesmo valor, senão o golden test da compactação fica instável.

## 7. Invariantes verificáveis

Cada linha é caso de teste obrigatório.

- `Assemble` chamada duas vezes com a mesma `Session` produz slices byte-a-byte idênticos.
- Anexar uma mensagem a `History` não altera nenhum byte do prefixo previamente produzido (RN-1). Verificado comparando o prefixo comum antes e depois.
- Nenhuma saída de `Assemble` contém dígito de timestamp, contador ou ID de sessão (RN-2).
- Nada altera `Session.Tools` depois da criação (RN-3) — varredura do repositório. O campo é público, então a garantia é um guarda, não o tipo: as definições ficam no prefixo cacheado, e mudá-las no meio da sessão faz todo turno seguinte pagar o prompt inteiro de novo, em silêncio.
- A ordem das seções é sempre a da tabela 4, para qualquer combinação de campos presentes ou ausentes.
- `Summary == nil` produz saída sem qualquer marcador de resumo.
- `Plan` nunca devolve `ToIdx` que separe `RoleAssistant` com `ToolCalls` dos seus `RoleTool`.
- `Plan` nunca inclui a última `RoleUser` nem nada posterior a ela no trecho compactado (RN-6).
- `Estimate` é determinística para a mesma entrada.
- `Fraction` e `BandFor` são determinísticas para a mesma entrada.
- `BandFor` é monotônica: fração maior nunca devolve faixa menor.
- O limiar mais alto de `Band` é estritamente menor que `CompactAt` — asserção sobre as constantes.
- Nenhuma saída de `Assemble` contém a fração nem a faixa (RN-2) — varredura.
- Travessia de faixa emite **uma vez**; permanecer acima do limiar não reemite.
- Compactação rearma a faixa anunciada; subir de novo emite de novo.
- A cauda preservada tem **dois pisos** e vence o que proteger mais: `KeepTurns` em contagem de turnos e `KeepFraction` em tokens da janela. A RN-6 fica acima dos dois.
- Sem janela reportada, a fração fica de fora e a contagem decide: sem denominador não há fração, e proteger tudo pararia a compactação de acontecer.
- `Assemble` não realiza I/O: verificado por teste que falha se o pacote importar `os`, `net` ou `time` fora de tipos.

## 8. Changelog

- [202608102200 — Orçamento de contexto realimentado](changelog/202608102200-orcamento-de-contexto-realimentado.md)
- [202608110900 — Chaves declaradas e não lidas saem da spec](changelog/202608110900-chaves-declaradas-e-nao-lidas.md)
- [202608250000 — O contexto avisa antes de cortar](changelog/202608250000-o-contexto-avisa-antes-de-cortar.md)
- [202608250100 — A cauda tem dois pisos](changelog/202608250100-a-cauda-tem-dois-pisos.md)
- [202609151700 — Compactar por pedido](changelog/202609151700-compactar-por-pedido.md)
- [202610180100 — A spec segue o código](changelog/202610180100-a-spec-segue-o-codigo.md)
