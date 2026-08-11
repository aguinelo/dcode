# Planning: Adaptador de Provider

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608072334-provider-adapter.r.spec.md`.

## 1. Nível de estabilidade

**`experimental`.** Vive em `internal/provider` no MVP. Quando uma terceira família existir, avaliar promoção para `pkg/provider` como ponto de extensão público — aí passa a `stable` e ganha critério de promoção próprio.

## 2. Os dois eixos

Um `Provider` é a **composição** de um transporte e uma família (RN-1). Nenhum dos dois é utilizável sozinho.

```go
// Package: internal/provider

// Transport é o formato de fio. Reusável entre famílias, não carrega
// adaptação nem limiar.
type Transport interface {
    Name() string // "openai" | "anthropic"

    // Do envia a requisição já serializada pela família e devolve os
    // eventos brutos do fio.
    Do(ctx context.Context, wire WireRequest) (<-chan WireEvent, error)
}

// Family é a adaptação. Carrega os limiares dos contratos comportamentais
// e os limites padrão do turno (RN-9).
type Family interface {
    Name() string // "minimax-m3" | "claude"

    // Transports lista os formatos de fio compatíveis, o primeiro sendo o
    // preferido. Uma família pode falar mais de um dialeto — é o caso do
    // MiniMax M3, e a razão de os dois eixos existirem.
    Transports() []string

    // Models lista os prefixos de modelo que esta família reivindica.
    Models() []string

    Window(model string) (int, error)

    // DefaultLimits são os limites de turno adequados a esta família (RN-9).
    DefaultLimits() Limits

    // Encode serializa o contexto neutro no corpo esperado pelo transporte.
    Encode(req Request, transport string) (WireRequest, error)

    // NewDecoder cria o decoder de UM stream.
    //
    // Decodificar não pode ser função pura de um frame: os argumentos de uma
    // tool call chegam partidos entre frames, e a call só existe inteira
    // quando o stream diz que terminou. Como ela é partida é específico do
    // dialeto, então o decoder pertence à família.
    NewDecoder(tools []contextpkg.ToolDef) Decoder
}

// Decoder traduz os frames brutos de um stream em eventos neutros, validando
// tool call contra o schema declarado (RN-8).
//
// Com estado e de uso único: um por stream, nunca compartilhado. Devolve zero
// ou mais eventos por frame — zero quando o frame trouxe só um fragmento,
// vários quando o fim do stream libera as calls que estavam sendo montadas.
type Decoder interface {
    Decode(ev WireEvent) ([]StreamEvent, error)
}

type Limits struct {
    MaxIterations int // teto de iterações do loop para esta família
    MaxOutputTokens int // 0 = default do provedor
}

// Provider é a composição. Construído pelo Registry, nunca à mão.
type Provider interface {
    Family() Family
    Transport() Transport
    Window(model string) (int, error)
    Limits() Limits
    Stream(ctx context.Context, req Request) (<-chan StreamEvent, error)
}

type Request struct {
    Model     string
    Messages  []contextpkg.Message // tipos neutros do dcode (RN-2)
    Tools     []contextpkg.ToolDef
    MaxTokens int
}
```

> `Encode` recebe o transporte como parâmetro porque uma família que fala dois dialetos serializa diferente em cada um. É exatamente o ponto que um eixo só não expressaria.

### 2.1 Famílias do MVP

| Família | `Models()` | `Transports()` | `MaxIterations` |
|---|---|---|---|
| `minimax-m3` | `MiniMax-M3`, `minimax-m3` | `openai`, `anthropic` | **200** |
| `claude` | `claude-` | `anthropic` | **50** |

**Por que 200 para M3 e 50 para Claude.** O default de 50 foi dimensionado pelo caso de um refactor cruzando dez arquivos. M3 é treinado para horizonte longo — a MiniMax demonstrou uma execução com 1.959 tool calls — e 50 truncaria trabalho legítimo. O detector de repetição em 3 chamadas idênticas continua sendo o mecanismo real contra loop patológico; o teto é backstop, e backstop acompanha o horizonte do modelo.

`minimax-m3` prefere `openai` porque é o dialeto com protocolo de tool-calling mais exercitado, e é o que os limiares medem.

**Família desconhecida não existe.** Modelo que não casa com nenhum `Models()` falha na criação da sessão, listando as famílias disponíveis. O escape hatch é explícito — `--family generic` — e emite aviso de que os limiares não foram medidos para aquele modelo.

## 3. Eventos de stream

```go
type StreamEventType string

const (
    EventTextDelta     StreamEventType = "text_delta"
    EventToolCall      StreamEventType = "tool_call"
    EventDone          StreamEventType = "done"
    EventError         StreamEventType = "error"
)

type StreamEvent struct {
    Type      StreamEventType
    Text      string                // EventTextDelta
    ToolCall  *contextpkg.ToolCall  // EventToolCall, já validado (ver abaixo)
    Usage     *Usage                // EventDone
    Err       *ProviderError        // EventError
}

type Usage struct {
    InputTokens       int
    OutputTokens      int
    CacheReadTokens   int // quanto do prefixo veio de cache — métrica-chave da ADR-03
    CacheWriteTokens  int
}
```

> `CacheReadTokens` não é telemetria decorativa. É a única medida direta de que o contexto append-only está funcionando. Se ele ficar próximo de zero em sessão longa, o motor de contexto regrediu.

> **O que "validado" quer dizer, exatamente.** `validateToolCall` confere **duas** coisas antes de a chamada chegar ao laço: que o nome está no conjunto declarado, e que os argumentos são JSON válido. Ela **não** confere o JSON Schema da ferramenta.
>
> A distinção não é acadêmica. É a diferença entre recusar `delete_file` — que é o que o limiar de `no-phantom-tool` mede, e é garantia estrutural — e garantir que `record_release` veio com o objeto aninhado preenchido, que **não** é garantido em lugar nenhum. Um cenário de `toolcall-schema-valid` que confiasse nesta linha mediria "o modelo devolveu JSON", pergunta que `{}` responde.
>
> Por isso o juiz daquele cenário decodifica a estrutura e exige os campos obrigatórios por conta própria. Validar schema no adaptador é decisão em aberto, não dívida escondida: custaria uma dependência de JSON Schema num projeto que escreveu o próprio leitor de TOML para não ter uma.

**Ordem garantida:** zero ou mais `EventTextDelta` e `EventToolCall` intercalados, terminados por exatamente **um** `EventDone` ou **um** `EventError`. Nunca ambos, nunca nenhum.

## 4. Classes de erro

```go
type ErrorClass string

const (
    ErrClassAuth        ErrorClass = "auth"          // credencial inválida ou ausente
    ErrClassQuota       ErrorClass = "quota"         // cota ou limite de gasto
    ErrClassRateLimit   ErrorClass = "rate_limit"    // pedido rápido demais; tem RetryAfter
    ErrClassContextSize ErrorClass = "context_size"  // contexto excede a janela
    ErrClassBadRequest  ErrorClass = "bad_request"   // requisição malformada; culpa nossa
    ErrClassToolSchema  ErrorClass = "tool_schema"   // tool call não valida (RN-8)
    ErrClassTransport   ErrorClass = "transport"     // rede, DNS, timeout
    ErrClassProvider    ErrorClass = "provider"      // 5xx do outro lado
    ErrClassCanceled    ErrorClass = "canceled"      // ctx cancelado
)

type ProviderError struct {
    Class      ErrorClass
    Message    string        // legível por humano; NUNCA contém credencial (RN-6)
    RetryAfter time.Duration // > 0 apenas em ErrClassRateLimit
    Retryable  bool
}

func (e *ProviderError) Error() string
```

**Mapa de decisão do loop** — é para isto que a classificação existe (RN-5):

| Classe | `Retryable` | O que o loop faz |
|---|---|---|
| `auth` | não | aborta o turno, erro ao usuário |
| `quota` | não | aborta o turno, erro ao usuário |
| `rate_limit` | sim | espera `RetryAfter`, repete a mesma requisição |
| `context_size` | sim | força compactação e repete |
| `bad_request` | não | aborta; é bug nosso, precisa aparecer |
| `tool_schema` | sim | devolve erro ao modelo como resultado de ferramenta (RN-8) |
| `transport` | sim | repete com recuo exponencial, até o teto |
| `provider` | sim | repete com recuo exponencial, até o teto |
| `canceled` | não | encerra silenciosamente |

## 5. Registro e resolução

```go
type Registry struct{ /* ... */ }

func (r *Registry) RegisterTransport(t Transport)
func (r *Registry) RegisterFamily(f Family)

// Resolve compõe o Provider: modelo → família → transporte.
// transportOverride vazio usa o preferido da família.
func (r *Registry) Resolve(model, transportOverride string) (Provider, error)
```

Ordem de resolução:

1. `model` casa com o `Models()` de exatamente uma família. Zero casamentos → erro listando as famílias disponíveis. Mais de um → erro de configuração; prefixos de família não podem se sobrepor.
2. Transporte é o `transportOverride`, se dado, ou o primeiro de `Transports()`.
3. Override que não está em `Transports()` da família → erro nomeando os compatíveis.

Nunca cai em família genérica por default. O default silencioso é o que entrega tool-calling ruim sem sinal.

## 6. Contratos comportamentais

> Seção presente porque o `.r` classifica o escopo como misto. Verificação por limiar, não por asserção. Ver `docs/conventions/SDD-HARNESS.pt-BR.md`, seção 4.

Mede a fidelidade da família de modelo, não a corretude do código.

| ID | Cenário | Comportamento esperado | Limiar | Fixture |
|---|---|---|---|---|
| `toolcall-schema-valid` | ferramenta com schema de objeto aninhado | tool call valida contra o schema na primeira tentativa | ≥ 97% | `testdata/evals/toolcall-schema-valid/` |
| `toolcall-recover` | tool call rejeitada por RN-8, erro devolvido | próxima tentativa corrige e valida | ≥ 90% | `testdata/evals/toolcall-recover/` |
| `no-phantom-tool` | prompt sugestivo de ferramenta inexistente | não inventa nome de ferramenta fora das declaradas | 100% | `testdata/evals/no-phantom-tool/` |

**Regras:**
- Modelo e versão de cada medição ficam no `.config.spec.md`. Trocar de modelo invalida o limiar, não o cenário.
- `no-phantom-tool` a 100% é legítimo: o adaptador **filtra** nome fora do conjunto declarado, então o limiar mede o filtro. Se algum dia o filtro sair, o cenário desce de regime.
- Rebaixar limiar no mesmo PR que o quebra exige entrada em `changelog/`.

## 7. Invariantes verificáveis

- Todo stream termina em exatamente um `EventDone` **ou** um `EventError`, nunca ambos, nunca nenhum.
- Cancelar `ctx` fecha o canal e emite `EventError` com `ErrClassCanceled`.
- Nenhum tipo específico de provedor cruza a fronteira do pacote (RN-2), verificado por teste de importação.
- Nenhuma credencial aparece em `ProviderError.Message`, em log ou em evento (RN-6) — teste injeta chave sentinela e varre toda a saída.
- Tool call que não valida contra o schema nunca chega ao consumidor como `EventToolCall` (RN-8).
- Todo teste da suíte padrão roda com a rede desligada (RN-4).
- `RetryAfter > 0` apenas em `ErrClassRateLimit`.
- Prefixos de `Models()` não se sobrepõem entre famílias; sobreposição é erro de inicialização.
- `Resolve` com transporte fora de `Transports()` da família devolve erro nomeando os compatíveis.
- A mesma família codificando para dois transportes distintos produz corpos distintos e ambos válidos — o teste que prova que os dois eixos são de fato ortogonais.
- `Limits()` devolve o default da família quando a configuração não sobrescreve.
- Tool call cujos argumentos chegam partidos entre frames é montada inteira antes de ser emitida.
- Duas tool calls paralelas no mesmo stream saem como duas calls, cada uma com seus argumentos.
- `finish_reason` repetido não emite a mesma call duas vezes.
- Call que o stream não terminou de emitir vira erro `tool_schema`, nunca execução.
- Raciocínio nunca aparece em evento de texto nem no histórico (RN-10).
- Frame que só carrega marcador de raciocínio e espaço em branco não produz evento algum.

## 8. Changelog

- [202608072352 — Transporte e família como eixos ortogonais](changelog/202608072352-transporte-familia-ortogonais.md)
- [202608082230 — Decode passa a ter estado por stream](changelog/202608082230-decode-com-estado-por-stream.md)
