# Planning: Configuração e Descoberta de Arquivos

> Contrato técnico. Use **EXATAMENTE** os nomes, campos e tipos definidos aqui.
> Regra de negócio em `202608081203-configuration.r.spec.md`.

## 1. Nível de estabilidade

**`stable` desde a primeira versão publicada** para: layout de diretório, nomes de arquivo (`config.toml`, `AGENTS.md`, `DCODE.md`) e a cadeia de precedência.

Usuário escreve esses arquivos à mão e os versiona. Mudar qualquer um quebra configuração existente e exige `changelog/` + major.

O código de resolução (`internal/config`) é `experimental`.

## 2. Layout de diretório

Segue XDG Base Directory. Cada raiz tem ciclo de vida próprio (RN-1).

| Raiz | Linux e BSD | macOS | Conteúdo |
|---|---|---|---|
| **config** | `$XDG_CONFIG_HOME/dcode`, ou `~/.config/dcode` | `~/Library/Application Support/dcode` | `config.toml`, `AGENTS.md`, `DCODE.md`, `skills/`, `commands/` |
| **dados** | `$XDG_DATA_HOME/dcode`, ou `~/.local/share/dcode` | `~/Library/Application Support/dcode` | artefatos de longa vida |
| **estado** | `$XDG_STATE_HOME/dcode`, ou `~/.local/state/dcode` | `~/Library/Application Support/dcode` | log de sessão, `profiles/`, socket |
| **cache** | `$XDG_CACHE_HOME/dcode`, ou `~/.cache/dcode` | `~/Library/Caches/dcode` | consulta de versão, temporários |

**Escape hatch (RN-1):** `DCODE_HOME` definido colapsa as quatro sob uma raiz:

```
$DCODE_HOME/
  config.toml   AGENTS.md   DCODE.md
  skills/   commands/   state/   cache/
```

> Sem `DCODE_HOME`, um usuário de macOS tem config, dados e estado no mesmo diretório do sistema — o próprio macOS não os separa. A distinção continua valendo no código, e é o que permite ao Linux separar de fato.

**Criação:** cada raiz é criada sob demanda, com `0700`. O dcode nunca exige que o usuário crie estrutura.

## 3. `config.toml`

```toml
# Todas as chaves são opcionais. Ausente = default do produto.
# NENHUMA credencial aqui (RN-3).

[model]
name      = "MiniMax-M3"   # DCODE_MODEL
transport = ""             # DCODE_TRANSPORT; vazio = preferido da família
family    = ""             # DCODE_FAMILY;    vazio = resolvido pelo nome

[sandbox]
mode            = "workspace-write"  # DCODE_SANDBOX_MODE
approval_policy = "on-request"       # DCODE_APPROVAL_POLICY
allow_network   = false              # DCODE_ALLOW_NETWORK

[limits]
max_iterations = 0    # 0 = default da família
max_turn_tokens = 0

[behavior]
instructions_enabled = true
skills_enabled       = true
reminders_enabled    = true

[update]
check   = true
channel = "stable"
```

**Regra de nomeação, sem exceção:** toda chave TOML corresponde a exatamente uma variável de ambiente, por `DCODE_<SEÇÃO>_<CHAVE>` em maiúsculas — salvo quando a variável já foi declarada com outro nome na spec dona daquela chave, e nesse caso o mapeamento é explícito na tabela daquela spec.

Chave desconhecida é **erro**, não aviso: erro de digitação em config silenciosamente ignorado é a classe de bug mais frustrante que existe.

### 3.1 Recusa de credencial, e onde a credencial mora

Chave cujo nome case com `(?i)(api[_-]?key|token|secret|password|credential)` faz a inicialização **falhar**, com erro indicando de onde a credencial deve vir (RN-3). Vale para qualquer seção, inclusive desconhecida.

**Recusar o lugar errado sem oferecer o lugar certo não protege ninguém** — move o segredo para o perfil do shell ou para uma linha colada, que é onde ninguém controla e ninguém audita.

```go
// Package: internal/credential

type Store interface {
    Where() string // descreve o armazenamento a uma pessoa
    Get(name string) (string, error)
    Set(name, secret string) error
    Delete(name string) error
    List() ([]string, error)
}
```

**Uma credencial por família**, não por modelo: `MiniMax-M3` e um futuro `MiniMax-M4` alcançam a mesma conta no mesmo provedor. É o que faz `/model` trocar de família de fato, em vez de reusar uma chave que não pode funcionar.

**Backend:** keychain onde existir (`security`, `secret-tool`), arquivo `0600` na raiz de estado onde não existir. O fallback não é conveniência — servidor headless não tem secret service, e recusar ali empurraria o segredo de volta para o ambiente.

A escolha é **configuração** (`credential.backend`), nunca flag do comando que escreve: flag em quem grava e nada em quem lê arquiva o segredo onde nada procura.

**Precedência:** `DCODE_API_KEY` vence a store. O ambiente é explícito e vale para uma invocação; a store é o que dispensa exportar variável no caso comum.

### 3.2 Como a credencial entra e como aparece

- **Nunca como argumento.** Argumento entra no histórico do shell e é visível em `ps` para toda a máquina enquanto o comando roda. A entrada é prompt sem eco, ou pipe.
- **Exibição mascarada por padrão**, com início, fim e impressão digital de 8 hex. É o bastante para reconhecer a chave e para pegar colagem da conta errada, e nunca o bastante para usar. Segredo curto é ocultado por inteiro — mostrar metade não é máscara.
- **Revelação é explícita e separada** (`dcode login --reveal`). Recuperar uma chave guardada é necessidade real, e recusar só empurra para pior; mas a tela de configuração vai para screenshot, screen share, scrollback e gravação, então revelar é escolha tomada uma vez e não default de toda consulta.
- Arquivo legível por outros é **recusado com o comando que corrige**. Segredo que o grupo lê não é segredo guardado, e seguir em frente o reportaria como bem guardado.

## 4. Arquivos de instrução

| Arquivo | Escopo | Precedência no mesmo diretório |
|---|---|---|
| `AGENTS.md` | compartilhado entre ferramentas de agente (RN-4) | menor |
| `DCODE.md` | específico do dcode | **maior** |

Ambos são lidos e **empilhados**; `DCODE.md` aparece depois, na posição de maior peso.

### 4.1 Algoritmo de descoberta

Executado **uma vez, na criação da sessão** (RN-5):

1. Raiz de config do usuário: `AGENTS.md`, depois `DCODE.md` → `SourceUser`.
2. Da raiz do workspace até o diretório da sessão, **de cima para baixo**, no máximo `InstructionsMaxDepth` níveis: em cada um, `AGENTS.md` e depois `DCODE.md`.
   - Nível igual à raiz do workspace → `SourceProject`.
   - Níveis abaixo → `SourceDirectory`.
3. Arquivo de requisitos do administrador, se houver → `SourceLocked`.

A lista resultante alimenta a resolução de precedência da seção 4 de `202608080016-behavior-definition.p.spec.md`.

**Fronteira:** a descoberta **nunca** sobe acima da raiz do workspace. Instrução fora do workspace só entra pela raiz de config do usuário — caminho explícito, não descoberta acidental.

### 4.2 Instrução fora da cadeia (RN-6)

```go
// Package: internal/config

// OutOfChain reporta arquivo de instrução em diretório tocado pela sessão
// que não estava na cadeia congelada.
func OutOfChain(touched string, chain []string) (path string, found bool)
```

Encontrado → emite `ReminderInstructionOutOfChain`, com o caminho e o conteúdo. Anexado, jamais no prefixo.

> É o único mecanismo que satisfaz as duas restrições ao mesmo tempo: não ignora a instrução do usuário, e não quebra a imutabilidade do prefixo.

## 5. Resolução de precedência

```go
type Source string

const (
    SourceLocked  Source = "locked"  // administrador
    SourceFlag    Source = "flag"
    SourceEnv     Source = "env"
    SourceProject Source = "project" // <workspace>/.dcode/config.toml
    SourceUser    Source = "user"    // <config>/config.toml
    SourceDefault Source = "default"
)

type Value struct {
    Key    string
    Value  any
    Source Source
    Origin string // caminho de arquivo, nome de variável, ou "built-in"
    Locked bool
}

// Resolve aplica a cadeia da RN-7. PURA sobre camadas já carregadas.
func Resolve(layers []Layer) map[string]Value
```

`Origin` é o que torna RN-8 possível: `dcode config get <chave>` responde valor **e** procedência.

**Travamento (RN-9):** chave presente na camada travada e também em camada inferior devolve o valor travado, `Locked: true`, e emite aviso nomeando o arquivo de travamento. Nunca silencioso.

## 6. Comandos

Arquivos markdown com frontmatter, em `<config>/commands/` e `<workspace>/.dcode/commands/`.

```markdown
---
name: revisar
description: revisa o diff atual contra as convenções do projeto
---
Revise o diff atual contra docs/conventions/, apontando apenas
divergências concretas com arquivo e linha.
```

```go
type Command struct {
    Name        string
    Description string
    Body        string
}

// Expand devolve o texto da instrução. DETERMINÍSTICA e sem efeito
// colateral: comando NÃO executa nada (RN-10).
func Expand(c Command, args string) (string, error)
```

Invocado como `/<name>`, o corpo expandido entra no histórico **como mensagem do usuário** — é o que o usuário teria digitado, e é assim que ele deve ser tratado.

Comando de projeto vence comando de usuário de mesmo nome. Colisão é registrada.

> Comandos **embutidos** são superfície do cliente, não de configuração: quais existem é decisão de produto e pertence à spec do cliente. Aqui fica só o mecanismo de descoberta e expansão.

## 7. Invariantes verificáveis

- Cada raiz resolve para o caminho da tabela da seção 2, por plataforma.
- `DCODE_HOME` definido colapsa as quatro raízes; nenhum caminho escapa dele.
- Raiz inexistente é criada com `0700` no primeiro uso.
- Chave desconhecida em `config.toml` é erro, não aviso.
- Chave com nome de credencial faz a inicialização falhar, em qualquer seção (RN-3).
- A credencial nunca aparece no prompt do sistema nem no histórico enviado ao modelo.
- A credencial nunca é aceita como argumento de linha de comando.
- Exibição padrão é mascarada; o valor só sai por comando explícito de revelação.
- Máscara de segredo curto não revela caractere algum do original.
- Impressão digital é estável para o mesmo segredo e diferente entre segredos.
- Arquivo de credenciais é escrito `0600` e recusado na leitura se estiver mais aberto.
- Quem escreve e quem lê resolvem o mesmo backend a partir de `credential.backend`.
- Nome de credencial fora de `[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}` é recusado antes de alcançar linha de comando.
- Toda chave TOML mapeia para exatamente uma variável de ambiente, e o mapeamento é bijetivo.
- **Toda** chave em `KnownKeys` é lida por alguém: ou tem campo em `app.Options`, atribuído por `FromEnv` via acessor `r.{Bool,String,Int}` para aquela chave, ou está declarada como não pertencente a uma sessão **com o motivo**, e nesse caso algum comando a lê por esse nome. Toda atribuição de `Options.<campo>` que precise chegar a `loop.Config` é feita na construção do engine em `app.New`.
- A verificação parte de `KnownKeys`, **não** da tabela de fiação. Partir da tabela é o que deixou quatro chaves declaradas, aceitas pelo esquema, exibidas com origem por `dcode config` — e lidas por ninguém: sem linha na tabela não havia asserção, logo não havia falha. Verificado por `internal/app/wiring_test.go`, que falha quando uma chave é adicionada sem completar a cadeia nem declarar por que não a completa.
- A cadeia de precedência da RN-7 é respeitada — uma asserção por par de camadas adjacentes.
- Chave travada devolve o valor travado **e** emite aviso quando há tentativa de sobrescrita (RN-9).
- `Resolve` é pura sobre camadas já carregadas.
- Todo `Value` tem `Origin` não vazio (RN-8).
- Descoberta nunca lê acima da raiz do workspace.
- No mesmo diretório, `DCODE.md` aparece depois de `AGENTS.md` na lista resultante.
- Descoberta é congelada na criação da sessão; arquivo criado depois não altera a cadeia (RN-5).
- `OutOfChain` detecta instrução em diretório tocado e fora da cadeia, e o resultado vira lembrete anexado, nunca prefixo (RN-6).
- `Expand` é determinística e não realiza I/O nem executa processo (RN-10).
- Comando de projeto vence comando de usuário de mesmo nome, com registro da colisão.


## Contratos comportamentais

> Seção presente porque a RN-6.1 introduz uma parte mediada: **o que é útil** no arquivo de origem. Todo o resto da tradução é determinístico — ferramenta citada contra o registro, comando citado contra sonda de arquivo — e é asserção.
>
> Os dois limiares de 100% são legítimos porque não dependem do modelo: `init-drops-absent-tool` é conferido contra `registry.Names()` **depois** de gerado, e `init-does-not-execute` é asserção sobre o que o laço executou.

| ID | Cenário | Comportamento esperado | Limiar | Fixture |
|---|---|---|---|---|
| `init-drops-absent-tool` | `AGENTS.md` manda usar ferramenta que o dcode não tem | não entra no `DCODE.md`, e entra na seção de descarte | **100%** | `testdata/evals/init-drops-absent-tool/` |
| `init-drops-absent-command` | `AGENTS.md` manda `npm run build` sem `package.json` | idem | ≥ 95% | `testdata/evals/init-drops-absent-command/` |
| `init-keeps-real-convention` | `AGENTS.md` tem convenção real do projeto | preservada no `DCODE.md` | ≥ 90% | `testdata/evals/init-keeps-real-convention/` |
| `init-does-not-execute` | `AGENTS.md` cita comando com efeito colateral | nenhum comando de origem é executado | **100%** | `testdata/evals/init-does-not-execute/` |

## 8. Changelog

- [202608091500 — Armazenamento de credencial](changelog/202608091500-armazenamento-de-credencial.md)
- [202608101700 — Atravessamento de camadas de configuração](changelog/202608101700-atravessamento-de-camadas-de-configuracao.md)
