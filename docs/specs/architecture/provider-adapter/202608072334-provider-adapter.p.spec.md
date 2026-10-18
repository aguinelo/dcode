# Planning: Adaptador de Provider

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608072334-provider-adapter.r.spec.md`.

## 1. Nível de estabilidade

**`experimental`.** Vive em `internal/provider`. Esta seção dizia para avaliar a promoção a `pkg/provider` "quando uma terceira família existir"; hoje existem quatro (`minimax-m3`, `claude`, `gemini`, `generic`) e a avaliação não foi feita. Fica registrado como pendência, não como decisão: o pacote continua interno e `experimental` até alguém decidir a promoção, que ganha critério próprio e passa a `stable`.

## 2. Os dois eixos

Um `Provider` é a **composição** de um transporte e uma família (RN-1). Nenhum dos dois é utilizável sozinho.

```go
// Package: internal/provider

// Transport é o formato de fio. Reusável entre famílias, não carrega
// adaptação nem limiar.
type Transport interface {
    Name() string // TransportOpenAI ("openai") | TransportAnthropic ("anthropic")

    // Do envia a requisição já serializada pela família e devolve os
    // eventos brutos do fio.
    Do(ctx context.Context, wire WireRequest) (<-chan WireEvent, error)
}

// Family é a adaptação. Carrega os limiares dos contratos comportamentais
// e os limites padrão do turno (RN-9).
type Family interface {
    Name() string // "minimax-m3" | "claude" | "gemini" | "generic"

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

    // AcceptsImages e AcceptsPDF dizem o que os modelos desta família leem
    // além de texto. Declarados, nunca tentados: o que uma família não declara
    // é recusado na submissão do turno, nomeando o modelo, em vez de virar
    // erro do provedor depois do envio. Texto não se declara — arquivo de texto
    // vai escrito na mensagem, e toda família lê texto. Só `claude` lê PDF.
    AcceptsImages() bool
    AcceptsPDF() bool

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

    // Close diz o que o fim do stream significa. Um dialeto pode terminar a
    // mensagem e continuar falando — o OpenAI repete `finish_reason` e põe o
    // uso no último frame —, então o decoder segura o evento terminal até o
    // transporte dizer que nada mais vem.
    Close() []StreamEvent
}

// WireRequest é o pedido já serializado pela família; o transporte só lê o
// envelope.
type WireRequest struct {
    Model  string
    Body   json.RawMessage
    Stream bool
}

// WireEvent é um frame bruto do fio. Data vazio com Done marca o fim; Err é
// falha do transporte (conexão, timeout, status).
type WireEvent struct {
    Data []byte
    Done bool
    Err  error
}

type Limits struct {
    MaxIterations int // teto de iterações do loop para esta família
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
| `minimax-m3` | `MiniMax-M3`, `minimax-m3` | `openai`, `anthropic` | **2.000** |
| `claude` | `claude-` | `anthropic` | **50** |
| `gemini` | `gemini-` | `openai` | **50** |
| `generic` | nenhum — só por `--family generic` / `DCODE_FAMILY` | `openai` | **5.000** |

| Família | Janela | Imagens | PDF | Medida? |
|---|---|---|---|---|
| `minimax-m3` | 1.000.000 fixo | sim | não | sim — os limiares foram medidos contra ela |
| `claude` | por modelo, tabela em `claude_models.go` | sim | sim | não — avisa (RN-11) |
| `gemini` | 1.000.000 fixo, palpite conservador | sim | não | um contrato, `boundary-full-access-acts`; não avisa |
| `generic` | 128.000 fixo, ou `DCODE_WINDOW` | não | não | não — aviso próprio, mais largo |

`gemini` fala o dialeto OpenAI pela superfície de compatibilidade do Google e
herda de `MiniMaxM3` a codificação e o decoder; o que sobrescreve é o eixo da
família — nome, modelos, janela, limites e o que lê. A superfície nativa
(`:streamGenerateContent`) seria outro transporte e não existe. `generic` não
reivindica prefixo nenhum (`Models()` vazio), então nunca é resolvida por
nome de modelo: só `ResolveFamily` a alcança (seção 5).

**Por que 2.000 para M3 e 50 para Claude.** O default de 50 foi dimensionado pelo caso de um refactor cruzando dez arquivos. M3 é treinado para horizonte longo — a MiniMax demonstrou uma execução com 1.959 tool calls — e 50 truncaria trabalho legítimo. O número foi 200 até esta citação ser levada a sério: o teto ficava em um décimo da execução que o justificava, e truncou uma real. O detector de repetição em 5 chamadas idênticas continua sendo o mecanismo real contra loop patológico; o teto é backstop, e backstop acompanha o horizonte do modelo.

**`generic` usa 5.000, não 50.** Herdava o mesmo default de `Claude` até este
número ser questionado — errado para o que essa família de fato serve: mais
frequentemente um modelo local, rodado especificamente pra fugir de um teto
de nuvem. Relatado em campo: um turno cortado em silêncio pelo teto, sem
nada na tela dizendo por quê — a metade do defeito que
`202609151400-o-turno-que-parava-mudo.md` corrige do lado do cliente; esta é
a outra metade, do lado do número. O detector de repetição continua sendo o
mecanismo real e não muda com este número; o teto aqui só limita custo e
tempo de parede, e um modelo local não paga o primeiro em dólar de API
nenhum.

**O teto de uma resposta também é da família, por modelo.** O dialeto anthropic exige `max_tokens`, e o que ia nele era um 8.192 fixo — um dezesseis avos do que a maioria dos modelos atuais aceita, cortando resposta longa num tamanho que ninguém escolheu. `Claude.OutputCeiling(model)` responde pelo limite publicado de cada linha de modelos (128.000 nos atuais, 64.000 em `claude-haiku-4-5` e na linha 4.5, 32.000 em `claude-opus-4`/`4-1`, 4.096 em `claude-3-haiku`), casado pelo prefixo mais longo. Modelo que a tabela não conhece recebe 32.000 (`claudeDefaultOutput`), não o maior número da tabela: teto acima do aceito é requisição recusada, teto abaixo é só resposta que para antes e diz. `minimax-m3` no dialeto anthropic segue com 8.192, que é o que sempre mandou. `DCODE_MAX_OUTPUT_TOKENS` (`202608072335-agent-loop.config.spec.md`) vence os dois. A parada no teto vira `Truncated` no `EventDone` — `stop_reason: max_tokens` num dialeto, `finish_reason: length` no outro.

**A janela também é por modelo, da mesma tabela.** `Claude.Window(model)` devolvia 200.000 fixo; todo modelo desde `claude-opus-4-6` tem 1.000.000, e a sessão era compactada a um quinto do que cabia. Linhas 4.5 e anteriores e `claude-haiku-4-5` seguem com 200.000, e modelo que a tabela não conhece também: janela pequena demais compacta cedo, grande demais é requisição recusada no meio do turno.

**Aliases sem sufixo casam por nome exato.** As linhas da geração 4 são
`claude-opus-4-0`, `claude-opus-4-2025`, `claude-sonnet-4-0` e
`claude-sonnet-4-2025`, e os aliases sem data nem `-0` — `claude-sonnet-4` e
`claude-opus-4` — não casavam com nenhuma: caíam nos defaults, e o teto de
`claude-sonnet-4` ficava em 32.000, metade dos 64.000 que o modelo aceita.
`claudeAliases` leva cada um à linha do modelo que nomeia (64.000 e 32.000 de
teto, 200.000 de janela). O casamento é exato, nunca por prefixo: como prefixo,
`claude-sonnet-4` daria o seu teto a toda Sonnet 4.x que a tabela não conhece.
Ver `changelog/202610172355-os-aliases-sem-data-tem-o-teto-publicado.md`.

`minimax-m3` prefere `openai` porque é o dialeto com protocolo de tool-calling mais exercitado, e é o que os limiares medem.

**Família desconhecida não existe.** Modelo que não casa com nenhum `Models()` falha na criação da sessão, listando as famílias disponíveis. O escape hatch é explícito — `--family generic` — e emite aviso de que os limiares não foram medidos para aquele modelo.

### 2.2 Perfis de modelo

Um perfil nomeia o pacote inteiro — modelo, família, transporte, endpoint,
janela — pra `/model <nome>` trocar tudo de uma vez, em vez de só o nome do
modelo. Vive em `models.toml` (`docs/specs/architecture/configuration`, §2),
lido com `ParseSections` em vez do parser bijetivo de `config.toml`: o nome do
perfil é dado que a pessoa escolheu, não uma chave que o produto declarou —
exatamente o caso que `ParseSections` existe pra atender, e o mesmo motivo por
que `requirements.toml` também é um arquivo à parte.

```toml
[profile.qwen-local]
model          = "qwen3.5-9b"
family         = "generic"
transport      = "openai"
base_url       = "http://192.168.0.149:1234/v1"
window         = 32000
max_iterations = 5000
```

Um perfil Anthropic pode também nomear o workspace que a chave dele exige,
`anthropic_workspace = "wrkspc_..."`, com o mesmo sentido de
`model.anthropic_workspace` (`.config`, seção 2). Ausente, o configurado vale.

`model` é obrigatório; o resto é opcional, com o mesmo default de sempre —
vazio deixa o prefixo do modelo resolver a família. Projeto **estende** usuário
por nome: um perfil do projeto com o nome de um do usuário é a resposta do
projeto, não uma colisão.

**A resolução é por sessão, não por processo.** `Daemon.build` resolve
`req.Model` contra os perfis carregados na inicialização do daemon antes de
tratá-lo como nome de modelo cru, através de `applyModelRequest` — função
pura, testada separadamente de `build`, pela mesma razão que
`Limits.withFamily` fica fora do laço do turno. Um nome que casa com um
perfil aplica o pacote inteiro (`Model`, `Family`, `Transport`, `BaseURL`,
`Window` e `MaxIterations`, cada um só quando maior que zero, e
`AnthropicWorkspace`, só quando não vazio); um nome que
não casa segue o caminho de sempre. Não há mudança de protocolo —
`CreateSessionRequest.Model` continua sendo uma string — porque a resolução
acontece inteiramente do lado do daemon, que já recarrega o `Provider` do
zero a cada sessão (RN de isolamento por sessão, `build`'s próprio
comentário: "each session gets its own... provider").

## 3. Eventos de stream

```go
type StreamEventType string

const (
    EventTextDelta        StreamEventType = "text_delta"
    EventReasoningDelta   StreamEventType = "reasoning_delta"    // pensamento do modelo; nunca entra no histórico (RN-10)
    EventToolCallOpened   StreamEventType = "tool_call_opened"   // uma call começou a chegar: nome e id já conhecidos
    EventToolCallProgress StreamEventType = "tool_call_progress" // quanto dos argumentos dela já chegou
    EventToolCall         StreamEventType = "tool_call"
    EventDone             StreamEventType = "done"
    EventError            StreamEventType = "error"
)

type StreamEvent struct {
    Type      StreamEventType
    Text      string                // EventTextDelta, EventReasoningDelta
    ToolCall  *contextpkg.ToolCall  // EventToolCall, já validado (ver abaixo)
    CallID    string                // EventToolCallOpened, EventToolCallProgress
    CallName  string                // EventToolCallOpened, EventToolCallProgress
    Bytes     int                   // EventToolCallProgress: bytes de argumento já chegados
    Usage     *Usage                // EventDone; e EventError cobrado mesmo assim — a recusa
    Err       *ProviderError        // EventError
    Truncated bool                  // EventDone: o provedor parou no teto de saída
}

type Usage struct {
    InputTokens       int
    OutputTokens      int
    CacheReadTokens   int // quanto do prefixo veio de cache — métrica-chave da ADR-03
    CacheWriteTokens  int
}
```

> **No dialeto anthropic o cache só existe se for pedido.** A família `claude` marca três pontos de cache (`cache_control: ephemeral`), abaixo dos quatro que a API aceita: a última ferramenta, o system e o último bloco da última mensagem. O provedor monta ferramentas, system e mensagens nessa ordem e guarda o prefixo até cada marca; como o contexto é append-only, a rodada seguinte reenvia esse prefixo sem mudança e o lê do cache. A marca não faz parte dos bytes guardados, então mudá-la para o novo último bloco a cada rodada não invalida prefixo nenhum. O system vira lista de blocos só quando carrega a marca. `minimax-m3` no mesmo dialeto não marca: nada diz que o endpoint dele lê a marca. O uso do prompt — entrada, leitura e escrita do cache — chega no `message_start`, e o decoder o guarda para o `EventDone`.

> `CacheReadTokens` não é telemetria decorativa. É a única medida direta de que o contexto append-only está funcionando. Se ele ficar próximo de zero em sessão longa, o motor de contexto regrediu.

> **O que "validado" quer dizer, exatamente.** `validateToolCall` confere **duas** coisas antes de a chamada chegar ao laço: que o nome está no conjunto declarado, e que os argumentos são JSON válido. Ela **não** confere o JSON Schema da ferramenta.
>
> A distinção não é acadêmica. É a diferença entre recusar `delete_file` — que é o que o limiar de `no-phantom-tool` mede, e é garantia estrutural — e garantir que `record_release` veio com o objeto aninhado preenchido, que **não** é garantido em lugar nenhum. Um cenário de `toolcall-schema-valid` que confiasse nesta linha mediria "o modelo devolveu JSON", pergunta que `{}` responde.
>
> Por isso o juiz daquele cenário decodifica a estrutura e exige os campos obrigatórios por conta própria. Validar schema no adaptador é decisão em aberto, não dívida escondida: custaria uma dependência de JSON Schema num projeto que escreveu o próprio leitor de TOML para não ter uma.

**Ordem garantida:** zero ou mais `EventTextDelta`, `EventReasoningDelta` e `EventToolCall` intercalados, terminados por exatamente **um** `EventDone` ou **um** `EventError`. Nunca ambos, nunca nenhum. `EventToolCallOpened` e `EventToolCallProgress` podem vir antes da `EventToolCall` que descrevem; quem os ignora vê exatamente a sequência de antes, e é isso que deixa um provedor que não sabe informá-los ficar calado em vez de fingir.

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
    ErrClassRefused     ErrorClass = "refused"       // o modelo recusou responder
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
| `context_size` | sim | força compactação e repete; sem o que compactar, ou com resumo que falha, aborta o turno, erro ao usuário |
| `bad_request` | não | aborta; é bug nosso, precisa aparecer |
| `tool_schema` | sim | devolve erro ao modelo como resultado de ferramenta (RN-8) |
| `transport` | sim | repete com recuo exponencial, até o teto |
| `provider` | sim | repete com recuo exponencial, até o teto |
| `canceled` | não | encerra silenciosamente |
| `refused` | não | encerra o turno em `refused`, erro ao usuário; nunca repete |

**De onde a classe sai quando o status não basta.** A Anthropic não responde
prompt longo demais com 413: responde 400 `invalid_request_error`, com
`prompt is too long` ou `exceed context limit` na mensagem. Esse 400 é
`context_size`; qualquer outro 400 continua `bad_request`, porque é bug nosso e
compactar o esconderia. E um erro pode chegar **dentro** de um stream que já
respondeu 200 — um frame `type: "error"`. Ele recebe a classe do seu par HTTP
(`overloaded_error` e `api_error` → `provider`, `rate_limit_error` →
`rate_limit`, `authentication_error` → `auth`, `billing_error` → `quota`,
`invalid_request_error` → `bad_request` ou `context_size`) e leva a mensagem do
provedor; tipo desconhecido vai para `provider`, nomeado.

**A recusa chega num 200.** O modelo que recusa responde com sucesso e termina
com `stop_reason: "refusal"` no `message_delta`, às vezes com `stop_details`
(`category`, `explanation`) — que é informativo e pode vir nulo; quem decide é
o `stop_reason`. A recusa pode vir antes de qualquer saída ou no meio do
stream, depois de texto já enviado. Ela termina o stream em `EventError` com
`refused`, não em `EventDone`: o texto anterior não é resposta, as chamadas de
ferramenta pendentes são descartadas e não liberadas, e o `EventError` leva o
`Usage`, porque a resposta recusada é cobrada como qualquer outra. A mensagem
diz que o modelo recusou, com a categoria e a explicação quando vieram.

## 5. Registro e resolução

```go
type Registry struct{ /* ... */ }

func NewRegistry() *Registry

func (r *Registry) RegisterTransport(t Transport)

// RegisterFamily recusa família cujo prefixo se sobrepõe ao de uma já
// registrada: prefixo ambíguo resolveria para quem chegou primeiro, calado.
func (r *Registry) RegisterFamily(f Family) error

// Resolve compõe o Provider: modelo → família → transporte.
// transportOverride vazio usa o preferido da família.
func (r *Registry) Resolve(model, transportOverride string) (Provider, error)

// ResolveFamily compõe com a família nomeada, sem resolução por prefixo. É o
// escape hatch (`--family generic`); quem chama avisa que os limiares não
// foram medidos.
func (r *Registry) ResolveFamily(familyName, transportOverride string) (Provider, error)

// FamilyFor diz qual família reivindica um modelo antes de existir Provider —
// a credencial, por exemplo, é resolvida por família e é com ela que o
// transporte é montado. Uma só implementação do casamento, para duas nunca
// discordarem sobre qual chave um modelo usa.
func FamilyFor(model string, families []Family) (Family, bool)

func (r *Registry) FamilyNames() []string    // ordenados, para mensagem de erro estável
func (r *Registry) TransportNames() []string // ordenados
```

Ordem de resolução:

1. `model` casa com o `Models()` de exatamente uma família. Zero casamentos → erro listando as famílias disponíveis. Mais de um não acontece: `RegisterFamily` já recusou a sobreposição na inicialização.
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
- Depois de cancelado o `ctx`, nada que o transporte ainda tinha é decodificado — nem frame à espera, nem o fechamento: o `select` não decide como um stream cancelado termina.
- Nenhum tipo específico de provedor cruza a fronteira do pacote (RN-2), verificado por teste de importação.
- Nenhuma credencial aparece em `ProviderError.Message`, em log ou em evento (RN-6) — teste injeta chave sentinela e varre toda a saída.
- Tool call que não valida contra o schema nunca chega ao consumidor como `EventToolCall` (RN-8).
- Frame que traz **uso e conteúdo juntos** entrega o conteúdo: o uso é lido depois das escolhas, nunca antes.
- Uso em frame separado continua terminando sem reemitir a chamada.
- Todo teste da suíte padrão roda com a rede desligada (RN-4).
- `RetryAfter > 0` apenas em `ErrClassRateLimit`.
- Prefixos de `Models()` não se sobrepõem entre famílias; sobreposição é erro de inicialização.
- `Resolve` com transporte fora de `Transports()` da família devolve erro nomeando os compatíveis.
- A mesma família codificando para dois transportes distintos produz corpos distintos e ambos válidos — o teste que prova que os dois eixos são de fato ortogonais.
- `Limits()` devolve o default da família quando a configuração não sobrescreve.
- Família cujo `Transports()` nomeia um dialeto **recusa** os outros no `Encode`; herdar a codificação não herda a declaração.
- Família **sem nenhuma medição** registrada avisa isso na sessão, e a lista de quem avisa é conferida contra as medições que existem, nos dois sentidos.
- Tool call cujos argumentos chegam partidos entre frames é montada inteira antes de ser emitida.
- Duas tool calls paralelas no mesmo stream saem como duas calls, cada uma com seus argumentos.
- `finish_reason` repetido não emite a mesma call duas vezes.
- Call que o stream não terminou de emitir vira erro `tool_schema`, nunca execução.
- Raciocínio nunca aparece em evento de texto nem no histórico (RN-10).
- Frame que só carrega marcador de raciocínio e espaço em branco não produz evento algum.
- O cabeçalho `anthropic-workspace-id` só sai no transporte `anthropic`, e só quando um workspace foi configurado.
- Requisição Anthropic sem teto configurado leva o teto que a família declara para o modelo, nunca um número fixo.
- Resposta que o provedor corta no teto de saída termina com `Truncated`, nos dois dialetos.
- Requisição `claude` marca no máximo três pontos de cache — a última ferramenta, o system e o último bloco da conversa —, verificado contra arquivo golden; `minimax-m3` não marca nenhum.
- O uso do `message_start` chega ao `EventDone`, com o que o cache leu e escreveu.
- Um 400 que diz que o prompt não cabe na janela vira `context_size` e força compactação; qualquer outro 400 continua `bad_request`.
- Frame de erro no meio de um stream Anthropic termina com a classe do seu par HTTP e a mensagem do provedor, nunca como `transport`.
- Resposta Anthropic com `stop_reason: "refusal"` termina em `EventError` da classe `refused`, sem repetição, sem liberar chamada pendente e com o uso cobrado.

## 8. Changelog

- [202608072352 — Transporte e família como eixos ortogonais](changelog/202608072352-transporte-familia-ortogonais.md)
- [202608082230 — Decode passa a ter estado por stream](changelog/202608082230-decode-com-estado-por-stream.md)
- [202609141900 — Um nome troca o pacote inteiro](changelog/202609141900-um-nome-troca-o-pacote-inteiro.md)
- [202609151500 — `generic` erra para o horizonte longo](changelog/202609151500-generic-erra-para-o-horizonte-longo.md)
- [202609281439 — Cancelado não é cara ou coroa](changelog/202609281439-cancelado-nao-e-cara-ou-coroa.md)
- [202609281513 — Uma linha, uma promessa](changelog/202609281513-uma-linha-uma-promessa.md)
- [202610151200 — Um turno carrega documentos](../client-server-protocol/changelog/202610151200-um-turno-carrega-documentos.md)
- [202610151200 — Uma chave sem workspace nomeia um](changelog/202610151200-uma-chave-sem-workspace-nomeia-um.md)
- [202610151300 — O teto de saída é do modelo](changelog/202610151300-o-teto-de-saida-e-do-modelo.md)
- [202610151400 — O prefixo Anthropic é lido do cache](changelog/202610151400-o-prefixo-anthropic-e-lido-do-cache.md)
- [202610151500 — A janela Claude é por modelo](changelog/202610151500-a-janela-claude-e-por-modelo.md)
- [202610161630 — O erro Anthropic diz a sua classe](changelog/202610161630-o-erro-anthropic-diz-a-sua-classe.md)
- [202610162300 — A recusa do modelo é dita](changelog/202610162300-a-recusa-do-modelo-e-dita.md)
- [202610171700 — A spec alcança o código](changelog/202610171700-a-spec-alcanca-o-codigo.md)
