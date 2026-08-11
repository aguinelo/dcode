# Implementing: Protocolo Client-Server do dcode

> Siga a ordem dos passos. Se algum passo aqui contradisser o `.r.spec.md`, **pare** — o `.r` tem precedência.

## Ordem de execução

### Passo 1 — Tipos do protocolo, sem servidor

`internal/protocol/`

- [ ] `Event`, `EventType` e as constantes de tipo da seção 5.1 do `.p`.
- [ ] `Session`, `SessionState`, `CreateSessionRequest`, `SubmitTurnRequest`.
- [ ] `ApprovalRequest`, `ApprovalDecision`, `ResolveApprovalRequest`.
- [ ] `Error` e as constantes de `code` da seção 8 do `.p`.
- [x] Testes de ida e volta de JSON para cada tipo, com golden file em `testdata/`. Regravados com `-update`; a nota `At` fixa evita o teste quebrar a cada execução.

> Pacote sem I/O e sem dependência de rede. É o vocabulário compartilhado — cliente e servidor importam daqui.

### Passo 2 — Log de eventos append-only

`internal/session/eventlog.go`

- [ ] `Append(ctx, sessionID, type, payload) (Event, error)` — atribui `Seq` sob lock, monotônico e sem lacunas.
- [ ] `Replay(ctx, sessionID, from uint64) (iter.Seq[Event], error)` — devolve `events_expired` abaixo da retenção.
- [ ] `Subscribe(ctx, sessionID, from uint64) (<-chan Event, error)` — reposição seguida de fluxo ao vivo, sem lacuna e sem duplicata na emenda.
- [x] Retenção conforme `DCODE_EVENT_RETENTION`; transbordo para disco conforme `DCODE_EVENT_SPILL`.

> Append-only, um JSON por linha, sem índice: replay é sequencial a partir de um número, que uma varredura responde direto, e a única operação que precisaria ser rápida é a que ninguém faz com frequência.
>
> Escreve **antes** de descartar. Descartar e escrever depois perderia os eventos em qualquer falha no meio, e buraco que o cliente não detecta é pior que replay recusado. Se a escrita falha, nada sai da memória.

**Teste obrigatório:** 1000 escritas concorrentes produzem `Seq` de 1 a 1000, sem repetição nem lacuna, sob `go test -race`.

> É a primitiva da RN-2. US-2, US-3 e US-6 caem toda dela — se este passo ficar frágil, os três quebram juntos.

### Passo 3 — Máquina de estados da sessão

`internal/session/session.go`

- [ ] Transições: `idle → running → idle`, `running → blocked → running`, qualquer → `closed`.
- [ ] Rejeitar entrada com turno ativo, devolvendo `turn_already_active` (RN-8).
- [ ] Turno sobrevive a zero clientes anexados (RN-1).
- [ ] `Interrupt` é idempotente e sempre resulta em `turn.completed` com `reason: "interrupted"`.

**Teste obrigatório:** transição inválida retorna erro, nunca faz `panic`.

### Passo 4 — Servidor HTTP sobre socket Unix

`internal/server/`

- [ ] Escutar em `DCODE_SOCKET` com permissão `0700`; detectar e remover socket órfão.
- [ ] Encerramento limpo: remove o socket, drena SSE, aguarda turnos em andamento com prazo.
- [ ] Endpoints de `Session` (criar, listar, detalhar, encerrar).
- [ ] `GET /health` e `GET /version` — marcados `stable`, então travar o formato com golden file agora.
- [ ] Mapeamento uniforme de erro para os `code` da seção 8.

### Passo 5 — Fluxo SSE

`internal/server/events.go`

- [ ] `GET /sessions/{id}/events?from=` com `id:` = `Seq` e `data:` = `Event`.
- [ ] Comentário `: ping` a cada 20s.
- [ ] Envio sempre em `select` com `ctx.Done()` — nunca escrita crua em canal (RN-3).
- [ ] Desconexão do cliente não cancela o turno.

**Teste obrigatório:** cliente que para de ler não bloqueia o agente; o turno chega a `turn.completed` mesmo assim.

### Passo 6 — Ciclo de aprovação

`internal/server/approvals.go` + `internal/sandbox/`

- [ ] Emitir `tool.approval_required` ao cruzar fronteira; sessão vai a `blocked`.
- [ ] `POST .../approvals/{id}` resolve; segunda chamada devolve `409` (RN-4).
- [ ] Prazo de `DCODE_APPROVAL_TIMEOUT` expira em **negação** (RN-5).
- [x] Aviso no boot quando `full-access` + `never`. `policy.BoundaryWarning` é pura, então servidor e cliente não precisam concordar sobre quando avisar. **Avisa, não recusa**: quem roda um container descartável quer exatamente essa combinação, e produto que discute ali é produto que se contorna — o que ele não pode é acontecer em silêncio.

**Teste obrigatório:** duas resoluções concorrentes — exatamente uma retorna `200`, exatamente uma retorna `409`.

> A avaliação da fronteira pertence à spec da ADR-02. Aqui só se implementa o transporte da decisão. Não duplicar a lógica de política neste pacote.

### Passo 7 — Cliente Go de referência

`pkg/client/`

- [ ] Cobrir todos os endpoints; `Subscribe` com reconexão automática a partir do último `Seq` + 1.
- [ ] Nenhum estado de sessão no cliente além da posição de leitura (RN-1).

> Primeiro consumidor do contrato. Se a API ficar desconfortável aqui, o problema é o protocolo — corrija a spec antes de seguir.

### Passo 8 — Verificação dos invariantes

`internal/server/invariants_test.go`

- [ ] Um teste para cada linha da seção 9 do `.p.spec.md`.
- [ ] Equivalência entre reposição e observação ao vivo, ignorando `At`.
- [ ] `go test -race ./...` limpo na CI.
- [ ] Build principal com `CGO_ENABLED=0` validado na CI.

### Passo 9 — Promover para `stable`

Só depois que **todos** valerem:

- [ ] Passos 1 a 8 concluídos.
- [ ] Uma segunda superfície consumindo o protocolo (o TUI não conta — foi escrito junto).
- [ ] Nenhuma mudança quebrando o contrato por 30 dias.
- [ ] Meta de RAM por sessão da ADR-01 medida e registrada como teste de regressão.

Ao promover: alterar a seção 1 do `.p.spec.md` para `stable` e criar entrada em `changelog/`.

## Ordem de dependência

```
Passo 1 (tipos)
  └─ Passo 2 (log de eventos)  ← primitiva; nada real funciona antes disto
       ├─ Passo 3 (máquina de estados)
       │    └─ Passo 6 (aprovação)   ← depende também da ADR-02
       └─ Passo 4 (servidor)
            └─ Passo 5 (SSE)
                 └─ Passo 7 (cliente)
                      └─ Passo 8 (invariantes)
                           └─ Passo 9 (promoção)
```

## Armadilhas conhecidas

- **`Seq` atribuído fora do lock** — origem clássica de lacuna sob concorrência. Atribuição e escrita no log são uma seção crítica só.
- **Emenda entre reposição e fluxo ao vivo** — se `Subscribe` abre o fluxo antes de terminar a reposição, aparece duplicata; se abre depois, aparece lacuna. Buffere o ao vivo durante a reposição e desduplique por `Seq`.
- **`defer` em laço de SSE** — acumula descritor. Fechar explicitamente.
- **`At` em golden file** — quebra o teste em toda execução. Zerar antes de comparar.
- **Alocação por `message.delta`** — é o caminho mais quente do sistema. Reusar buffer aqui, ou a meta de RAM da ADR-01 não se sustenta.

## Changelog

_Sem alterações desde a criação._
