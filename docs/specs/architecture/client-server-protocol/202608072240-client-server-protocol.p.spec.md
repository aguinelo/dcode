# Planning: Protocolo Client-Server do Harness

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608072240-client-server-protocol.r.spec.md`.

## 1. Nível de estabilidade

> Contrato público carrega nível declarado. Ver `docs/conventions/SDD-HARNESS.md`, seção 5.

| Nível | Significado |
|---|---|
| `experimental` | pode quebrar em qualquer versão, sem changelog |
| `stable` | quebra exige entrada em `changelog/` + incremento de major |
| `frozen` | não muda; só extensão aditiva |

**Estabilidade desta spec na v1: `experimental`.** Promove para `stable` no marco definido no `.i.spec.md`.

## 2. Transporte

| Item | Decisão |
|---|---|
| Camada | HTTP/1.1 sobre **socket de domínio Unix** |
| Corpo | JSON (`application/json`) |
| Fluxo | **SSE** (`text/event-stream`) |
| Prefixo de versão | `/v1` |
| Permissão do socket | `0700`, dono = usuário que iniciou o daemon |

**Por que HTTP+SSE e não gRPC:** sem etapa de codegen no build, alcançável direto por superfície web (desktop e IDE futuros), depurável com `curl --unix-socket`, e cliente HTTP existe em toda linguagem. gRPC seria mais rápido no fio, mas o gargalo aqui é o modelo, não a serialização — otimizar transporte seria otimizar o lugar errado.

## 3. Envelope de evento

```go
// Package: internal/protocol

type EventType string

type Event struct {
    Seq       uint64          `json:"seq"`        // monotônico por sessão, começa em 1
    SessionID string          `json:"session_id"`
    Type      EventType       `json:"type"`
    At        time.Time       `json:"at"`         // RFC 3339, precisão de ms
    Payload   json.RawMessage `json:"payload"`
}
```

- `Seq` é **por sessão**, nunca global. Sem lacunas, sem reuso.
- `At` é excluído de comparação em golden file — é o único campo não determinístico do envelope.
- `Payload` é objeto tipado por `Type`, definido na seção 5.

## 4. Endpoints

Todos sob `/v1`. Estabilidade individual declarada.

| Método | Caminho | Estab. | Descrição |
|---|---|---|---|
| `POST` | `/sessions` | `experimental` | Cria sessão. Devolve `Session`. |
| `GET` | `/sessions` | `experimental` | Lista sessões vivas. |
| `GET` | `/sessions/{id}` | `experimental` | Detalhe de uma sessão. |
| `DELETE` | `/sessions/{id}` | `experimental` | Encerra e libera a sessão. |
| `GET` | `/sessions/{id}/events` | `experimental` | **SSE.** Query `from` (uint64, default `1`). |
| `POST` | `/sessions/{id}/turns` | `experimental` | Submete entrada do usuário. `409` se já houver turno ativo (RN-8). |
| `POST` | `/sessions/{id}/interrupt` | `experimental` | Cancela o turno em andamento. Idempotente. |
| `POST` | `/sessions/{id}/approvals/{approval_id}` | `experimental` | Resolve pedido de permissão. `409` se já resolvido (RN-4). |
| `GET` | `/health` | `stable` | Liveness. Sem corpo além de `{"status":"ok"}`. |
| `GET` | `/version` | `stable` | Versão do servidor e do protocolo. |

## 5. Tipos

```go
type SessionState string

const (
    SessionStateIdle     SessionState = "idle"
    SessionStateRunning  SessionState = "running"
    SessionStateBlocked  SessionState = "blocked"  // aguardando aprovação
    SessionStateClosed   SessionState = "closed"
)

type Session struct {
    ID        string       `json:"id"`         // ULID
    State     SessionState `json:"state"`
    Workspace string       `json:"workspace"`  // caminho absoluto, raiz do sandbox
    Model     string       `json:"model"`
    CreatedAt time.Time    `json:"created_at"`
    LastSeq   uint64       `json:"last_seq"`
}

type CreateSessionRequest struct {
    Workspace   string `json:"workspace"`              // absoluto; obrigatório
    Model       string `json:"model,omitempty"`
    SandboxMode string `json:"sandbox_mode,omitempty"` // ver .config.spec.md
}

type SubmitTurnRequest struct {
    Text string `json:"text"`
}

type ApprovalDecision string

const (
    ApprovalAllow        ApprovalDecision = "allow"
    ApprovalAllowSession ApprovalDecision = "allow_session" // permite pelo resto da sessão
    ApprovalDeny         ApprovalDecision = "deny"
)

type ResolveApprovalRequest struct {
    Decision ApprovalDecision `json:"decision"`
}

type Error struct {
    Code    string         `json:"code"`    // estável, legível por máquina
    Message string         `json:"message"` // PT-BR, legível por humano
    Details map[string]any `json:"details,omitempty"`
}
```

### 5.1 Tipos de evento

| `type` | Payload | Quando |
|---|---|---|
| `session.created` | `Session` | criação |
| `turn.started` | `{"turn_id":string}` | entrada aceita |
| `message.delta` | `{"turn_id":string,"text":string}` | fragmento de texto do modelo |
| `tool.requested` | `{"turn_id":string,"tool_call_id":string,"name":string,"input":object}` | modelo pediu ferramenta |
| `tool.approval_required` | `ApprovalRequest` | cruzou fronteira do sandbox (RN-4) |
| `tool.approval_resolved` | `{"approval_id":string,"decision":ApprovalDecision}` | decisão registrada |
| `tool.completed` | `{"tool_call_id":string,"ok":bool,"output":string,"truncated":bool}` | ferramenta terminou |
| `turn.completed` | `{"turn_id":string,"reason":string}` | `reason`: `done`, `interrupted`, `error` |
| `session.compacted` | `{"from_seq":uint64,"to_seq":uint64}` | compactação de contexto (ADR-03) |
| `session.error` | `Error` | falha não atribuível a um turno |

```go
type ApprovalRequest struct {
    ApprovalID      string    `json:"approval_id"`
    TurnID          string    `json:"turn_id"`
    ToolCallID      string    `json:"tool_call_id"`
    Tool            string    `json:"tool"`
    Command         string    `json:"command,omitempty"`          // comando renderizado, quando houver
    BoundaryCrossed string    `json:"boundary_crossed"`           // "network" | "workspace_write" | "filesystem_read"
    ExpiresAt       time.Time `json:"expires_at"`
}
```

## 6. Fluxo de aprovação

Implementa RN-4 e RN-5, ligando ADR-02 a ADR-04.

1. Modelo pede ferramenta → `tool.requested`.
2. Executor avalia a política contra a fronteira do sandbox.
3. Dentro da fronteira → executa sem perguntar.
4. Cruzando a fronteira → sessão vai a `blocked`, emite `tool.approval_required`, **o turno para**.
5. Qualquer cliente anexado envia `POST .../approvals/{approval_id}`.
6. Primeira resposta vence; as demais recebem `409 approval_already_resolved`.
7. Emite `tool.approval_resolved`; sessão volta a `running`.
8. Sem resposta até `ExpiresAt` → **negado** (RN-5), emite `tool.approval_resolved` com `deny`.

> A fronteira é aplicada pelo sistema operacional (ADR-02); este protocolo só transporta a decisão. Um cliente **nunca** amplia o sandbox — só responde dentro da política já vigente.

## 7. Semântica do fluxo SSE

- Cada evento SSE tem `id:` igual ao `Seq` e `data:` com o `Event` serializado.
- `from` menor que o primeiro `Seq` retido → `410 events_expired`.
- Servidor envia comentário `: ping` a cada 20s para manter proxies e conexões ociosas vivas.
- Desconexão não afeta a sessão (RN-1). Reconectar com `from = último Seq recebido + 1`.
- Sem `Last-Event-ID`: a reposição é sempre explícita via `from`, para que o comportamento seja idêntico entre reconexão e primeira conexão.

## 8. Códigos de erro

| `code` | HTTP | Significado |
|---|---|---|
| `session_not_found` | 404 | id inexistente ou já encerrada |
| `turn_already_active` | 409 | viola RN-8 |
| `approval_already_resolved` | 409 | outro cliente decidiu antes |
| `approval_expired` | 410 | prazo esgotado; já negado |
| `events_expired` | 410 | `from` anterior à retenção |
| `workspace_invalid` | 400 | caminho não absoluto ou inacessível |
| `max_sessions_reached` | 503 | limite do `.config.spec.md` |
| `internal` | 500 | falha não classificada |

## 9. Invariantes verificáveis

Toda linha aqui é caso de teste obrigatório em `go test`. Ver seção 2 do `.r.spec.md`.

- `Seq` é estritamente crescente e sem lacunas, por sessão.
- Reproduzir de `from=1` gera exatamente a mesma sequência de eventos que a observação ao vivo, ignorando `At`.
- Nenhum evento é emitido após `turn.completed` do mesmo `turn_id`, exceto de outro turno.
- Sessão em `blocked` não emite `message.delta`.
- Cliente desanexado durante turno não altera a sequência final de eventos.
- Duas resoluções concorrentes da mesma aprovação: exatamente uma retorna `200`, a outra `409`.
- Aprovação expirada produz exatamente um `tool.approval_resolved` com `deny`.

## 10. Changelog

_Sem alterações desde a criação._
