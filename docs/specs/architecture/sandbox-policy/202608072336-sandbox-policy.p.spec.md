# Planning: Sandbox e Política de Permissão

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608072336-sandbox-policy.r.spec.md`.

## 1. Nível de estabilidade

**`experimental`.** Vive em `internal/sandbox` e `internal/policy`. Os nomes de modo (`read-only`, `workspace-write`, `full-access`) e de política (`untrusted`, `on-request`, `never`) são **`stable` desde já** — aparecem em config de usuário e em `CreateSessionRequest` do protocolo. Renomeá-los quebra configuração existente.

## 2. Os dois eixos

```go
// Package: internal/policy

type SandboxMode string

const (
    ModeReadOnly       SandboxMode = "read-only"
    ModeWorkspaceWrite SandboxMode = "workspace-write"
    ModeFullAccess     SandboxMode = "full-access"
)

type ApprovalPolicy string

const (
    PolicyUntrusted ApprovalPolicy = "untrusted" // escalona tudo que não é leitura no workspace
    PolicyOnRequest ApprovalPolicy = "on-request" // escalona só no cruzamento de fronteira
    PolicyNever     ApprovalPolicy = "never"      // nunca pergunta; nega o que cruzaria
)
```

> `PolicyNever` **nega** o que cruzaria a fronteira; não permite. Sem cliente para perguntar, a alternativa a negar seria conceder em silêncio, que viola RN-3.

## 3. Avaliação

```go
type Request struct {
    Tool      string
    Paths     []Access // caminhos tocados, já resolvidos
    Network   bool
    Command   string   // renderizado, quando houver execução de comando
}

type Access struct {
    Path  string // absoluto, com symlink resolvido (RN-4)
    Write bool
}

type Decision string

const (
    DecisionAllow    Decision = "allow"    // dentro da fronteira, sem perguntar
    DecisionEscalate Decision = "escalate" // cruza; pedir aprovação
    DecisionDeny     Decision = "deny"     // impossível no modo atual
)

type Verdict struct {
    Decision Decision
    Boundary string // "network" | "workspace_write" | "filesystem_write" | "filesystem_read"
    Reason   string // inglês, legível por humano
}

// Evaluate é PURA: sem I/O, sem relógio. A resolução de caminho acontece
// ANTES, em Resolve, para que a decisão seja exatamente testável.
func Evaluate(r Request, mode SandboxMode, pol ApprovalPolicy, workspace string) Verdict
```

### 3.1 Tabela de decisão

| Operação | `read-only` | `workspace-write` | `full-access` |
|---|---|---|---|
| Leitura dentro do workspace | allow | allow | allow |
| Leitura fora do workspace | escalate | escalate | allow |
| Escrita dentro do workspace | **deny** | allow | allow |
| Escrita fora do workspace | **deny** | escalate | allow |
| Rede | **deny** | escalate | allow |

Depois, a política filtra:

| Verdict de modo | `untrusted` | `on-request` | `never` |
|---|---|---|---|
| `allow` de escrita no workspace | escalate | allow | allow |
| `allow` (demais) | allow | allow | allow |
| `escalate` | escalate | escalate | **deny** |
| `deny` | deny | deny | deny |

> `untrusted` escalona a escrita no workspace, que `on-request` libera. É a diferença prática entre os dois.

## 4. Resolução de caminho

```go
// Resolve faz I/O: expande symlink e normaliza. Separado de Evaluate
// justamente para manter a decisão pura e testável.
func Resolve(path string, workspace string) (Access, error)
```

**Regras:**
- Caminho relativo resolve contra o workspace, nunca contra o diretório do processo.
- Symlink é resolvido até o alvo final; symlink apontando para fora **é** cruzamento (RN-4).
- `..` é normalizado antes da comparação.
- Caminho inexistente resolve o diretório-pai mais próximo que existe — criar arquivo novo é operação legítima.
- Comparação de contenção é por componente de caminho, **nunca** por prefixo de string: `/home/user/proj2` não está contido em `/home/user/proj`.

## 5. Fronteira do sistema operacional

```go
// Package: internal/sandbox

type Sandbox interface {
    // Wrap devolve o comando a executar, já envolvido pelo mecanismo do SO.
    Wrap(ctx context.Context, cmd Command, mode SandboxMode, workspace string) (*exec.Cmd, error)
    // Available informa se o mecanismo pode ser estabelecido nesta máquina.
    Available() error
}
```

| Plataforma | Mecanismo | cgo |
|---|---|---|
| macOS | `sandbox-exec` com perfil gerado | **não** |
| Linux | `bwrap` com namespaces de usuário | **não** |
| Windows | Windows Sandbox | **não** (MVP: não suportado) |

**Todos por `exec` de binário externo, sem cgo.** Preserva `CGO_ENABLED=0` e a compilação cruzada, conforme a ADR-01. A API nativa do Seatbelt via cgo é otimização futura, não requisito — e adiá-la é o que torna a fronteira viável no MVP.

**Falha fechada (RN-3):** `Available()` falhando aborta a criação da sessão com erro nomeando o binário ausente e como instalá-lo. Nunca degrada para execução sem fronteira.

## 6. Invariantes verificáveis

- `Evaluate` é pura: mesma entrada, mesmo `Verdict`, sem I/O.
- Toda combinação das tabelas 3.1 tem teste — são 15 células de modo mais 12 de política.
- `read-only` nunca devolve `allow` para operação de escrita, sob nenhuma política.
- `never` nunca devolve `escalate`.
- `Resolve` com symlink apontando para fora devolve `Access` fora do workspace.
- `/home/user/proj2` não é considerado contido em `/home/user/proj` — teste explícito de prefixo de string.
- `..` escapando do workspace é cruzamento, não erro.
- Nenhuma execução ocorre sem `Evaluate` prévio (RN-6), verificado com avaliador espião no executor.
- `Available()` falhando impede a criação da sessão; nenhum caminho executa sem fronteira.
- Config travada por administrador não é sobrescrita por variável de ambiente nem por flag (RN-7).

## 7. Changelog

_Sem alterações desde a criação._
