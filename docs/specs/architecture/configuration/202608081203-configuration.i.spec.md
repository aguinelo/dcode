# Implementing: Configuração e Descoberta de Arquivos

> Siga a ordem. Se algum passo contradisser o `.r.spec.md`, **pare** — o `.r` tem precedência.
> Todo código novo nasce de teste que falhou primeiro (`docs/conventions/TESTING.pt-BR.md`).

## Desbloqueia

Concluído o Passo 3, `202608080016-behavior-definition` deixa de ter pendência de caminho: o localizador injetado passa a ter implementação real, sem refactor.

## Ordem de execução

### Passo 1 — Resolução de raízes

`internal/config/config.go`

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

`internal/config/toml.go`

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

`internal/config/config.go`

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

`internal/config/config.go`

- [ ] Cadeia resolvida na criação da sessão e congelada (RN-5).
- [ ] `OutOfChain` detectando instrução em diretório tocado e fora da cadeia.
- [ ] Resultado emite `ReminderInstructionOutOfChain`, anexado.

**Testes obrigatórios:**
- Arquivo de instrução criado após a criação da sessão **não** altera a cadeia nem o prefixo.
- Tocar diretório com instrução fora da cadeia produz exatamente um lembrete.
- O lembrete nunca aparece no prefixo — varredura da saída de `behavior.Build`.

> É o passo que reconcilia duas restrições que parecem incompatíveis: não ignorar a instrução do usuário e não quebrar a imutabilidade do prefixo.

### Passo 5 — Cadeia de precedência

`internal/config/config.go`

- [ ] `Source`, `Value`, `Layer`, `Resolve` puro.
- [ ] Ordem exata da RN-7.
- [ ] `Origin` preenchido em todo `Value`.
- [x] Travada devolve o valor travado, `Locked: true` e **aviso** na tentativa de sobrescrita.

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

### Passo 7.5 — Tradução e reindex

> Acrescentado por [202608101900](changelog/202608101900-traducao-de-instrucoes-de-terceiros.md). Depende do Passo 3; independente dos Passos 5, 6 e 7.

`internal/config/translate.go` — a verificação, que é código, não prompt:

- [x] `VerifyTools(text string, have []string) []Finding` — ferramenta citada conferida contra o registro do produto. Fato, não julgamento.
- [x] `ProbeCommands(text string, fs fs.FS) []Finding` — comando citado conferido por **presença de arquivo** (`package.json`, `go.mod`, `Makefile`).
- [x] `SourceDigest(fsys fs.FS, files []string) string` e `RenderDigest`, com digest **por arquivo** — digest das origens, gravado no `DCODE.md` gerado.
- [x] `Diverged(dcodeMD string, fs fs.FS) ([]string, bool)` — nomeia **quais** arquivos mudaram desde a geração.

**Testes obrigatórios:**
- Guarda de importação: `translate.go` **não** importa `os/exec`. Comando de origem nunca é executado (RN-6.1).
- `VerifyTools` acusa ferramenta ausente do registro e não acusa ferramenta presente.
- `ProbeCommands` acusa `npm run build` sem `package.json` e não acusa com ele.
- `Diverged` devolve o nome do arquivo alterado, não só um booleano — o aviso precisa dizer **o que** mudou (RN-6.2).
- Nenhum caminho escreve por cima de `DCODE.md` existente.

`internal/tui/commands.go`:

- [x] `InitPrompt` reescrito: traduzir, não resumir. Exige a seção de descarte no arquivo gerado.
- [x] Aviso de início de sessão pelo canal de **lembrete**, nunca no prefixo.

> A guarda de importação é o mesmo desenho do Passo 7, e pelo mesmo motivo: "só rodo pra ver se funciona" é a tentação óbvia, e aqui ela executa instrução de repositório de terceiro dentro do workspace.

> `InitPrompt` é turno de modelo e portanto mediado. A verificação **não** está nele: roda sobre o resultado, em código. Prompt pedindo para conferir não é conferência.

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
- **Verificação escrita como instrução no `InitPrompt`** — pedir ao modelo que confira não é conferir. A verificação é código que roda sobre o resultado, ou não existe.
- **Reindex regenerando sozinho** — apaga a edição manual que o usuário fez no `DCODE.md` depois da geração, e some sem erro. Detecta, nomeia o arquivo, propõe.
- **`Diverged` devolvendo só booleano** — produz o aviso inútil "algo mudou". O usuário precisa do nome para saber se importa.
- **Aviso de instrução não traduzida virando bloqueio** — torna o produto inutilizável em repositório recém-clonado, que é o caso mais comum de todos.

## Changelog

- [202608101900 — Tradução de instruções de terceiros](changelog/202608101900-traducao-de-instrucoes-de-terceiros.md)
