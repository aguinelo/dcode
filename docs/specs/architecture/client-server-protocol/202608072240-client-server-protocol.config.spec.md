# Config: Protocolo Client-Server do dcode

> Nenhuma variável de ambiente nova pode aparecer no código sem estar aqui.
> Precedência: flag de linha de comando > variável de ambiente > arquivo de config > default.

## 1. Transporte e estado

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_SOCKET` | caminho | `$XDG_RUNTIME_DIR/dcode.sock`, ou `$TMPDIR/dcode-$UID.sock` se ausente | Caminho do socket de domínio Unix. Criado com `0700`. Removido no encerramento limpo; socket órfão é detectado por tentativa de conexão e removido. |
| `DCODE_STATE_DIR` | caminho | `$XDG_STATE_HOME/dcode`, ou `~/.local/state/dcode` | Raiz de logs de sessão e dados persistentes. |
| `DCODE_MAX_SESSIONS` | inteiro | `64` | Teto de sessões vivas. Ao exceder, `POST /sessions` responde `503 max_sessions_reached`. Existe para tornar a densidade de sessão da ADR-01 mensurável, não para limitar o usuário. |

## 2. Retenção de eventos

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_EVENT_RETENTION` | inteiro | `10000` | Eventos mantidos em memória por sessão para reposição. Abaixo do menor `Seq` retido, `GET /events?from=` responde `410 events_expired`. |
| `DCODE_EVENT_SPILL` | booleano | `true` | Quando `true`, eventos além da retenção vão para disco em `DCODE_STATE_DIR` e a reposição continua funcionando. Quando `false`, são descartados — só use em ambiente efêmero. |

## 3. Aprovação e sandbox

Valores espelham a ADR-02. O protocolo transporta a decisão; a fronteira é aplicada pelo sistema operacional.

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_SANDBOX_MODE` | enum | `workspace-write` | `read-only`, `workspace-write`, `danger-full-access`. Default por sessão; sobrescrevível em `CreateSessionRequest.SandboxMode`. |
| `DCODE_APPROVAL_POLICY` | enum | `on-request` | `untrusted`, `on-request`, `never`. Política de escalonamento, **ortogonal** ao modo de sandbox. |
| `DCODE_APPROVAL_TIMEOUT` | duração | `120s` | Prazo de `ApprovalRequest.ExpiresAt`. Esgotado, a decisão é **negar** (RN-5). `0` desativa o prazo — a sessão fica bloqueada indefinidamente; só use em depuração. |

> `DCODE_SANDBOX_MODE=danger-full-access` combinado com `DCODE_APPROVAL_POLICY=never` remove toda fronteira. O servidor **deve** registrar aviso em nível `warn` na inicialização quando essa combinação estiver ativa.

## 4. Runtime Go

Consequência direta da ADR-01: o custo aceito da escolha por Go é pressão de GC sob swarm.

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `GOMEMLIMIT` | tamanho | não definido | Teto flexível de memória do processo. Recomendado em ambiente com muitas sessões; sem ele o GC só reage a `GOGC`. |
| `GOGC` | inteiro | `100` (padrão do Go) | Reduzir para `50` troca CPU por menor pico de heap — avaliar sob carga real de swarm, não por palpite. |
| `DCODE_MAX_TOOL_OUTPUT` | tamanho | `256KB` | Limite de saída de ferramenta antes de truncar. Alimenta `tool.completed.truncated`. Existe para evitar OOM por saída não confiável. |

## 5. Observabilidade

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_LOG_LEVEL` | enum | `info` | `debug`, `info`, `warn`, `error`. Log estruturado via `log/slog`. |
| `DCODE_LOG_FORMAT` | enum | `text` | `text`, `json`. `json` para coleta automatizada. |
| `DCODE_TRACE_EVENTS` | booleano | `false` | Quando `true`, registra cada evento emitido em nível `debug`. Verboso; apenas depuração de protocolo. |

## 6. Constantes não configuráveis

Fixas em código, documentadas porque afetam o contrato observável.

| Constante | Valor | Motivo |
|---|---|---|
| Intervalo de ping SSE | `20s` | Abaixo do timeout de inatividade comum em proxies. |
| Prefixo de versão | `/v1` | Muda apenas em incremento de major. |
| Permissão do socket | `0700` | Implementa RN-6; não afrouxar sem spec de autenticação. |
| Formato de ID de sessão | ULID | Ordenável por tempo e seguro para nome de arquivo. |

## 7. Changelog

_Sem alterações desde a criação._
