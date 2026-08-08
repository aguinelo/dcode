# Implementing: Configuração e Descoberta de Arquivos

> Siga a ordem. Se algum passo contradisser o `.r.spec.md`, **pare** — o `.r` tem precedência.
> Todo código novo nasce de teste que falhou primeiro (`docs/conventions/TESTING.pt-BR.md`).

## Desbloqueia

Concluído o Passo 3, `202608080016-behavior-definition` deixa de ter pendência de caminho: o localizador injetado passa a ter implementação real, sem refactor.

## Ordem de execução

### Passo 1 — Resolução de raízes

`internal/config/paths.go`

- [ ] Quatro raízes por plataforma, conforme a tabela da seção 2 do `.p`.
- [ ] `DCODE_HOME` colapsa todas; nenhum caminho escapa.
- [ ] Criação sob demanda com `0700`.

**Testes obrigatórios, com `HOME` e variáveis XDG controlados:**
- Cada raiz resolve para o caminho esperado no Linux e no macOS.
- `DCODE_HOME` definido colapsa as quatro — varredura provando que nenhuma escapa.
- Raiz inexistente é criada com `0700`.
- Variável XDG vazia cai no fallback, não em caminho relativo.

> Primeiro porque tudo depois precisa saber onde procurar. Testar com `HOME` controlado, nunca contra o diretório real do desenvolvedor.

### Passo 2 — `config.toml` e recusa de credencial

`internal/config/file.go`

- [ ] Parsing TOML mapeado no esquema da seção 3 do `.p`.
- [ ] Chave desconhecida é erro quando `ConfigStrict`.
- [ ] **Recusa de credencial** por padrão de nome, em qualquer seção, inclusive desconhecida.
- [ ] Arquivo ausente não é erro.

**Testes obrigatórios:**
- Chave com nome de credencial faz a inicialização falhar, e o erro diz de onde a credencial deve vir.
- O teste cobre seção desconhecida — é o caminho por onde um segredo entraria despercebido.
- Chave desconhecida é erro em modo estrito, aviso fora dele.
- Toda chave TOML mapeia para exatamente uma variável de ambiente, e o mapeamento é bijetivo — varredura das duas direções.

> A recusa de credencial é a regra com maior retorno deste componente. Config é feita para ser versionada; aceitar segredo ali produz o vazamento mais comum que existe.

### Passo 3 — Descoberta de instruções

`internal/config/discover.go`

- [ ] Algoritmo da seção 4.1 do `.p`, de cima para baixo.
- [ ] `AGENTS.md` antes de `DCODE.md` no mesmo diretório.
- [ ] `SourceProject` na raiz do workspace, `SourceDirectory` abaixo dela.
- [ ] Teto de `InstructionsMaxDepth`.
- [ ] **Nunca** lê acima da raiz do workspace.

**Testes obrigatórios, com árvore real em `t.TempDir()`:**
- Monorepo com instrução em três níveis produz a cadeia na ordem certa.
- Arquivo acima da raiz do workspace **não** é lido, mesmo existindo.
- `DCODE.md` aparece depois de `AGENTS.md` no mesmo diretório.
- Symlink apontando para fora do workspace não contorna a fronteira.

> 🎯 **Aqui a spec de comportamento deixa de depender de decisão pendente.**

### Passo 4 — Congelamento e instrução fora da cadeia

`internal/config/chain.go`

- [ ] Cadeia resolvida na criação da sessão e congelada (RN-5).
- [ ] `OutOfChain` detectando instrução em diretório tocado e fora da cadeia.
- [ ] Resultado emite `ReminderInstructionOutOfChain`, anexado.

**Testes obrigatórios:**
- Arquivo de instrução criado após a criação da sessão **não** altera a cadeia nem o prefixo.
- Tocar diretório com instrução fora da cadeia produz exatamente um lembrete.
- O lembrete nunca aparece no prefixo — varredura da saída de `behavior.Build`.

> É o passo que reconcilia duas restrições que parecem incompatíveis: não ignorar a instrução do usuário e não quebrar a imutabilidade do prefixo.

### Passo 5 — Cadeia de precedência

`internal/config/resolve.go`

- [ ] `Source`, `Value`, `Layer`, `Resolve` puro.
- [ ] Ordem exata da RN-7.
- [ ] `Origin` preenchido em todo `Value`.
- [ ] Travada devolve o valor travado, `Locked: true` e **aviso** na tentativa de sobrescrita.

**Testes obrigatórios:**
- Uma asserção por par de camadas adjacentes — cinco pares.
- Todo `Value` tem `Origin` não vazio.
- Sobrescrita de chave travada emite aviso nomeando o arquivo de travamento.
- `Resolve` é puro sobre camadas já carregadas.

### Passo 6 — `dcode config get`

`cmd/dcode/config.go`

- [ ] Responde valor efetivo **e** procedência (RN-8).
- [ ] Indica travamento quando houver.

> Junto de `DCODE_DOCTRINE_DUMP`, forma o par de auditoria do produto: um mostra o que vai ao modelo, o outro de onde veio cada configuração. Sem os dois, o suporte vira "no meu funciona".

### Passo 7 — Comandos

`internal/config/commands.go`

- [ ] Descoberta em `<config>/commands` e `<workspace>/.dcode/commands`.
- [ ] Parsing de frontmatter.
- [ ] `Expand` determinística, **sem I/O e sem executar processo**.
- [ ] Projeto vence usuário em nome igual; colisão registrada.

**Testes obrigatórios:**
- `Expand` é determinística para a mesma entrada.
- Guarda: o pacote de expansão não importa `os/exec`.
- Colisão de nome resolve para o de projeto e registra.

> A guarda de importação existe porque a tentação de fazer comando executar algo aparece cedo. Execução fora do avaliador de política está proibida pela RN-6 da spec de sandbox, e comando não abre exceção.

### Passo 8 — Invariantes

`internal/config/invariants_test.go`

- [ ] Um teste por linha da seção 7 do `.p.spec.md`.
- [ ] `go test -race ./internal/config/...` limpo.
- [ ] Cobertura ≥ 90%.

## Ordem de dependência

```
Passo 1 (raízes)
  ├─ Passo 2 (config.toml)
  │    └─ Passo 5 (precedência)
  │         └─ Passo 6 (config get)
  ├─ Passo 3 (descoberta)        ← 🎯 desbloqueia behavior-definition
  │    └─ Passo 4 (congelamento e lembrete)
  └─ Passo 7 (comandos)
       └─ Passo 8 (invariantes)
```

## Armadilhas conhecidas

- **Testar contra o `HOME` real** — passa na máquina do desenvolvedor e quebra na CI, ou pior, escreve no diretório de quem roda o teste. Sempre `HOME` controlado.
- **`DCODE_HOME` esquecido em uma raiz** — a que escapar cria diretório fora da raiz colapsada, e o usuário descobre por acidente meses depois.
- **Recusa de credencial só nas seções conhecidas** — o segredo entra pela seção desconhecida, que é exatamente onde ninguém olha.
- **Chave desconhecida virando aviso por default** — erro de digitação em config é silenciosamente ignorado, e a regra que o usuário acha que está valendo nunca esteve.
- **Descoberta subindo acima do workspace** — lê instrução de projeto vizinho em monorepo aninhado, e o comportamento fica inexplicável.
- **Cadeia recalculada por turno** — invalida o prefixo e anula a ADR-03 sem erro visível.
- **`Expand` chamando `os/exec`** — cria segunda superfície de execução fora da política. A guarda de importação do Passo 7 existe para isso.
- **Config sem `Origin`** — inviabiliza RN-8, e o suporte perde a única ferramenta que resolve "no meu funciona".

## Changelog

_Sem alterações desde a criação._
