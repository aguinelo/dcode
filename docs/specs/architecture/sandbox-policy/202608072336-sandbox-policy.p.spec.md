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
    Boundary Boundary // tipada, nao string: um valor invalido nao compila
    Rule     string   // o padrao que escalou, para a aprovacao poder dizer POR QUE
    Reason   string // inglês, legível por humano
}

// Evaluate é PURA: sem I/O, sem relógio. A resolução de caminho acontece
// ANTES, em Resolve, para que a decisão seja exatamente testável.
func Evaluate(r Request, mode SandboxMode, pol ApprovalPolicy,
    rules Rules, inWorkspace func(Access) bool) Verdict
```

> **Correção de assinatura.** Esta seção nunca foi atualizada pelo changelog `202608091700`, que acrescentou as regras por caminho e por comando. `Evaluate` recebe `Rules` e recebe a contenção como **função**, não o workspace como string — porque resolver caminho faz E/S e a decisão não pode fazer, e é essa separação que torna a tabela inteira testável por asserção, uma por célula.
>
> `Resolve` é método de `Resolver` e não função livre, pelo mesmo motivo: a única parte que toca disco fica num tipo, e o resto do pacote continua puro.


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

## 3.1 Regras: atenção, nunca contenção

Dentro do workspace tudo é uniforme para o sandbox — e ele está certo: escrever
em `src/` e em `.git/hooks/` é o mesmo tipo de escrita. **Não é o mesmo tipo de
consequência**, e é para isso que servem as regras.

```go
type Rules struct {
    ConfirmWrite   []string // caminhos que perguntam antes de escrever
    ConfirmRead    []string // ler manda o conteúdo ao provedor do modelo
    ConfirmCommand []string // comandos que pausam
}
```

**Elas não contêm.** Padrão de comando é contornado sem esforço — `bash -c`, um
alias, um script — e padrão de caminho só enxerga o caminho que a ferramenta
declara. Quem contém é o sandbox. Ler regra como fronteira é a única forma de
ficar pior do que não tê-las, e é por isso que as fronteiras se chamam
`rule:write`, `rule:read` e `rule:command`: ninguém confunde com o que o SO
aplicou.

**Ordem e limites:**

1. Containment decide primeiro. Regra só é avaliada sobre o que o sandbox já
   permitiria — **nunca resgata** o que foi negado, ou a fronteira viraria
   negociável por configuração.
2. Regra só é avaliada onde alguém vai ser perguntado. Com `never` não há
   pessoa, logo não há pergunta — e transformar pergunta impossível em negação
   faria `never` ser **mais** restritivo que `on-request`, ao contrário do nome.
3. Escrita antes de leitura: chamada que faz as duas é escrita, que tem a cauda
   mais longa.
4. Caminho fora do workspace não tem forma relativa e regra nenhuma o alcança —
   containment já respondeu.

**Padrão** curto de propósito. Cada entrada entra por ser diferente **em
natureza** de um arquivo de código, não por ser importante:

| Padrão | Por quê |
|---|---|
| `.git/**` | escrita em `hooks/` roda no próximo commit, **fora do sandbox**, como o usuário |
| `.dcode/**` | configura o agente; agente que edita a própria configuração amplia o próprio alcance |
| `.env`, `*.pem`, `id_rsa`, `.npmrc`… | **ler** manda o conteúdo ao provedor do modelo, para fora da máquina |

Lista configurada **substitui** o default; quem escreveu uma lista disse o que
quer que pergunte, e manter a nossa por baixo faria da configuração dele uma
mentira. Lista vazia é como se diz "nada".

**Escopo da aprovação:** `allow session` é lembrado contra a **regra que casou**,
não contra o caminho exato. Editar três arquivos sob um diretório é uma decisão,
não três — e três perguntas é como se aprende a aprovar sem ler. Sem regra, a
chave continua sendo ferramenta + comando exato, que é o conservador: comando de
shell é opaco, e "o mesmo tipo de comando" não é algo que isto saiba julgar.

## 4. Resolução de caminho

```go
// Resolve faz I/O: expande symlink e normaliza. Separado de Evaluate
// justamente para manter a decisão pura e testável.
func (r *Resolver) Resolve(path string, write bool) (Access, error)
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
- Regra nunca transforma negação em pergunta.
- Regra avaliada sob política `never` nunca vira pergunta.
- `never` **nega** o que uma regra escalonaria: com ninguém para perguntar, a autorização expressa não chega.
- Rede concedida deixa de ser pergunta; retirada a concessão, volta a ser.
- Concessão de rede é autorização, nunca contenção — não abre o que o modo fechou.
- Comando destrutivo pede confirmação por default, em qualquer modo, e a regra que disparou é nomeada.
- Comando que sai da máquina pede confirmação: a contenção não vai junto.
- Trabalho comum de repositório não pede, inclusive `git push`, que alcança um remoto sem ser execução remota.
- Trabalho comum não pergunta: build, teste e commit passam sem escalonar.
- Regra dispara em `full-access`: os eixos são ortogonais e regra vive no de aprovação.
- Escrita é perguntada antes de leitura quando a mesma chamada faz as duas.
- Regra escrita contra o workspace não alcança caminho fora dele.
- A aprovação carrega o padrão que casou; consentir a regra que não se vê é consentir a nada.
- `allow session` é chaveado pela regra quando houve regra, e pelo comando exato quando não houve.
- Padrão em branco não casa com nada, e nunca com tudo.
- As regras efetivas são inspecionáveis por `--config`, com procedência.
- Workspace sob `/tmp` continua visível dentro do sandbox: o `tmpfs` é montado antes dele, nunca por cima.

- Rede concedida não entrega socket unix: no macOS o perfil libera tráfego IP e o resolvedor de nomes, nunca `(allow network*)`.
- Rede concedida inclui escutar: uma suíte que não abre porta não roda.
- Caminho nomeado como não legível não é lido de dentro do sandbox, e o mesmo caminho sem ser nomeado continua legível.
- `full-access` não esconde nada: modo que não promete fronteira não mantém uma escondida.
- Nada nomeado esconde os cofres de credencial mesmo assim; o home inteiro nunca é um nome válido.
- A própria credencial do dcode está entre os escondidos por default.
- O nome do arquivo de credencial escondido acompanha o que o cofre escreve.
- Socket concedido por nome é alcançável; o não concedido continua fora.
- Caminho concedido é gravável fora do workspace, e `read-only` não ganha nenhum.
- A chave privada entra nos escondidos assim que o agente é alcançável, e não antes.
- Token de agente sem agente rodando concede nada, nunca a string vazia.
- Socket unix é alcançável exatamente onde já se pode escrever — e `read-only`, que não escreve em lugar nenhum, não alcança nenhum.
- Socket de runtime de contêiner é coberto no Linux, e um socket real deixa de ser socket dentro do sandbox.
- `full-access` mantém a concessão ampla: modo que não promete fronteira não finge estreitá-la.

- `workspace-write` concede os diretórios que uma toolchain precisa para compilar: cache e temporário do usuário.
- Nenhuma regra concede o diretório home; o que é concedido é nomeado um a um.
- `read-only` não concede nenhum deles — o modo significa o mesmo nos dois backends.
- Ambiente ausente ou nulo não concede nada, e não derruba a sessão.
- Backend só se declara disponível depois de **provar** que consegue aplicar uma fronteira, nos dois sistemas.
- Fronteira indisponível faz o teste **pular dizendo o motivo**, nunca falhar como se o trabalho estivesse errado.
## 7. Changelog

- [202608091700 — Regras por caminho e por comando](changelog/202608091700-regras-por-caminho-e-comando.md)
- [202608150300 — Workspace sob /tmp](changelog/202608150300-workspace-sob-tmp.md)
- [202608190030 — Trabalho comum não pergunta, destruição pergunta sempre](changelog/202608190030-trabalho-comum-nao-pergunta.md)
- [202608190130 — Uma toolchain alcança o próprio cache](changelog/202608190130-a-toolchain-alcanca-o-proprio-cache.md)
- [202608190230 — Uma fronteira aninhada é detectada, não adivinhada](changelog/202608190230-uma-fronteira-aninhada-e-detectada.md)
