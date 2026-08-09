# Três temas em aberto

Pauta de discussão, não plano de execução. Cada tema vira uma spec RPI quando
for decidido — este documento existe para que a decisão seja tomada com o
estado real em mãos, e não de memória.

Português, como as specs: é material pré-spec e vira spec.

**Estado medido em 2026-08-09**, sobre `09c1a46`.

---

# Tema 1 — Liberar a escrita, com confirmação por diretório e por comando

## O que existe hoje

O default do produto **já é escrita**: `sandbox.mode = workspace-write` e
`sandbox.approval_policy = on-request`. O que estava em `read-only` era o script
de teste do scratchpad, não o produto.

A fronteira atual é **binária**:

| Situação | Hoje |
|---|---|
| Escrever dentro do workspace | passa, sem perguntar |
| Escrever fora do workspace | pergunta |
| Ler qualquer lugar | passa |
| Rede | depende de `allow_network`; se fechada, o SO bloqueia e nada é perguntado |
| Comando específico | **não existe regra alguma** |

Não há allowlist, denylist, nem qualquer regra por caminho ou por comando.
`policy.Evaluate` decide por modo e por containment, e mais nada.

## O que está sendo pedido

Liberar de fato — "o editor criar vida e codificar" — mas com confirmação em
**diretórios e comandos específicos**. Isso é uma camada de regras que não
existe.

## As perguntas a decidir

**1. O que dispara confirmação, se hoje escrever no workspace já não dispara?**

O incômodo provavelmente não é escrever em `src/`. É escrever em coisas que
parecem workspace mas não são de mexer: `.git/`, `.env`, `node_modules/`,
`go.sum`, arquivos de lock, o próprio `.dcode/`.

> Ponto que vale decidir explicitamente: **`.dcode/` do workspace configura o
> agente.** Um agente que pode editar a própria configuração pode ampliar o
> próprio alcance. Isso é diferente em natureza de editar `src/`.

**2. Regra por comando: lista de bloqueio ou lista de liberação?**

- **Bloqueio** (`rm -rf`, `git push`, `curl | sh`): flui por padrão, pergunta no
  que é reconhecidamente perigoso. Fácil de conviver, e **fácil de furar** —
  `bash -c 'rm -rf …'`, um alias, um script. Uma regra que casa texto de comando
  é uma regra que o modelo contorna sem querer.
- **Liberação** (`go`, `git status`, `ls`, `make`): pergunta em tudo que não
  está na lista. Não fura, e atrapalha muito no começo.

Minha leitura: **lista de bloqueio dá falsa sensação de segurança.** O sandbox é
o que segura de fato — foi ele que impediu a escrita fora do workspace no teste,
não o prompt. Regra por comando serve para *chamar atenção*, não para conter, e
vale escrever isso na spec para ninguém confundir os dois papéis depois.

**3. Onde as regras moram?**

`config.toml` do usuário, do projeto, ou o arquivo travado por administrador?
Se a regra mora no `.dcode/` do projeto, e o agente pode escrever no projeto, a
regra é auto-editável. Ver pergunta 1.

**4. "Não perguntar de novo" tem escopo de quê?**

Hoje `A` (allow session) é chaveado por `ferramenta + comando exato`, o que para
shell é quase inútil: `go test ./a` e `go test ./b` são duas perguntas. As
opções são por ferramenta, por regra que casou, ou por prefixo — e cada uma
concede mais do que a anterior.

## O que eu recomendaria discutir primeiro

Separar dois problemas que estão juntos na mesma frase:

- **Atrito**: hoje quase não existe mais, depois da correção do `09c1a46`.
  Vale rodar com o padrão antes de decidir, para não projetar regra sobre um
  incômodo que já saiu.
- **Proteção de caminhos sensíveis**: `.git/`, `.env`, `.dcode/` — esse é real e
  independente do atrito.

---

# Tema 2 — Instalar e atualizar localmente a cada build

## O que existe hoje

- `make build` → `./bin/dcode`. Nada mais.
- **Não há alvo `install`** no Makefile.
- `dcode update` existe, mas baixa release assinado do GitHub — não serve para
  build local, e não deveria servir.
- Rodar hoje é `./bin/dcode` ou o script do scratchpad, que é a gambiarra citada.

## O que está sendo pedido

`dcode` no PATH, atualizado a cada build, sem cerimônia.

## As perguntas a decidir

**1. Onde instalar?**

`$HOME/.local/bin` é o mesmo default do `install.sh` — consistente, e não pede
privilégio. `go install` já funciona e põe em `$GOBIN`; o problema é que ele não
injeta versão via ldflags, então o binário reporta `dev`.

**2. O binário local deve se identificar como local?**

Recomendo **sim, enfaticamente**: algo como `0.1.0-dev+a91f2c4-dirty`. Um
binário local que se apresenta igual a um release publicado é como um relato de
bug vira uma hora perdida. E `dcode update` precisa **recusar** substituir um
binário local, ou o trabalho não commitado some.

**3. Qual o comando?**

`make install` é o óbvio. Vale também `make dev` que instala e já roda? A
diferença é se você quer o binário no PATH ou só executar o que acabou de
compilar.

> Este é o tema mais barato dos três e o que mais melhora seu dia a dia. Não tem
> decisão de arquitetura, só escolha de default.

---

# Tema 3 — Configuração de modelo e credenciais

## O que existe hoje

| Peça | Estado |
|---|---|
| Escolher modelo | `DCODE_MODEL` ou `model.name` no `config.toml` |
| Transporte/dialeto | `DCODE_TRANSPORT`, resolvido pela família se vazio |
| Base URL | `DCODE_BASE_URL` |
| **Credencial** | **só `DCODE_API_KEY`, variável de ambiente** |
| Guardar credencial | **não existe** |
| `config.toml` com credencial | **recusa a iniciar** (por desenho) |
| Comando de setup / login | **não existe** |
| Keychain, keyring, secret storage | **não existe** |

O `config.toml` recusa qualquer chave que case com
`api[_-]?key|token|secret|password|credential`, em qualquer seção — incluindo
seções desconhecidas. Isso está certo: o arquivo é feito para ser versionado e
sincronizado.

## O problema real

**Recusamos o lugar errado sem oferecer o lugar certo.** A consequência é que a
chave vai para o `.zshrc`, ou é colada num terminal — que foi exatamente o que
aconteceu nesta sessão: a chave acabou no transcript da conversa.

Refusar sem oferecer alternativa não protege ninguém; só move o segredo para um
lugar que não controlamos e não auditamos.

## As perguntas a decidir

**1. Onde a credencial deve morar?**

- **Keychain do SO** (`security` no macOS, `secret-tool`/libsecret no Linux):
  o certo. Custa um backend por plataforma e uma degradação clara onde não
  existir.
- **Arquivo `0600` na raiz de estado**, separado do `config.toml`: bem mais
  simples, e explicitamente *não* no arquivo que se versiona. Protege contra o
  vazamento comum (commit, sync), não contra leitura local.
- **Só ambiente**, como hoje: nada a implementar, e o problema continua.

**2. Como a credencial entra?**

Um `dcode login` que **lê de stdin sem eco** e nunca aceita a chave como
argumento — argumento entra no histórico do shell e aparece em `ps`. Se for
implementado, isso é invariante de spec, não detalhe.

**3. Existe setup guiado?**

Um `dcode init` ou primeiro-uso que pergunta modelo, transporte e credencial, e
escreve o `config.toml` sem o segredo. Hoje o primeiro uso é ler o README e
exportar variável.

**4. O que a interface mostra?**

Hoje nada indica se há credencial configurada até um turno falhar com `auth`.
Um `/config` que diga *que existe uma credencial e de onde ela veio*, sem nunca
mostrar valor, fecha o par de auditoria que `--dump-prompt` já abriu.

**5. Vários provedores ao mesmo tempo?**

`/model` já troca de modelo abrindo sessão nova. Se dois modelos exigirem
credenciais diferentes, uma chave só não basta — e isso muda o formato do que
for guardado. Vale decidir antes de escrever o arquivo, não depois.

## Invariantes que eu proporia, decidamos o resto como for

1. Credencial **nunca** em arquivo versionável — a recusa atual fica.
2. Credencial **nunca** como argumento de linha de comando.
3. Credencial **nunca** em log, evento, erro ou prompt — já é RN-6 do provider,
   e já há redação implementada.
4. `--dump-prompt` e `/config` podem dizer que existe e de onde veio; **jamais**
   o valor.

---

# Ordem que eu sugeriria

1. **Tema 2** — barato, sem decisão de arquitetura, melhora hoje.
2. **Tema 3** — é o que tem um problema de segurança real e presente.
3. **Tema 1** — o maior, e o que mais se beneficia de você rodar o padrão atual
   antes, para separar atrito percebido de proteção que falta.
