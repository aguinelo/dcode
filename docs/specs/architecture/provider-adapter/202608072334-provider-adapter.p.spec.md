# Planning: Adaptador de Provider

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608072334-provider-adapter.r.spec.md`.

## 1. Nível de estabilidade

**`experimental`.** Vive em `internal/provider` no MVP. Quando uma terceira família existir, avaliar promoção para `pkg/provider` como ponto de extensão público — aí passa a `stable` e ganha critério de promoção próprio.

## 2. A interface

```go
// Package: internal/provider

type Provider interface {
    // Name identifica a família, não o endpoint (RN-1).
    Name() string

    // Window devolve o tamanho da janela de contexto do modelo em tokens (RN-7).
    Window(model string) (int, error)

    // Stream envia o contexto e devolve um canal de eventos.
    // O canal fecha ao fim do turno do modelo ou ao cancelamento de ctx.
    // Erro de transporte chega como StreamEvent do tipo EventError, não como
    // retorno — o consumidor lê tudo por um caminho só.
    Stream(ctx context.Context, req Request) (<-chan StreamEvent, error)
}

type Request struct {
    Model    string
    Messages []contextpkg.Message // tipos neutros do harness (RN-2)
    Tools    []contextpkg.ToolDef
    MaxTokens int // 0 = default do provedor
}
```

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
    ToolCall  *contextpkg.ToolCall  // EventToolCall, já validado contra schema
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

## 5. Registro de famílias

```go
type Registry struct{ /* ... */ }

func (r *Registry) Register(p Provider)
func (r *Registry) Resolve(model string) (Provider, error)
```

Resolução por prefixo de nome de modelo, declarado por cada adaptador. Modelo desconhecido devolve erro explícito na criação da sessão — nunca cai em adaptador genérico por default, porque um default silencioso entrega tool-calling ruim sem sinal.

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

## 8. Changelog

_Sem alterações desde a criação._
