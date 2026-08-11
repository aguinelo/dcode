# Implementing: Sandbox e Política de Permissão

> Siga a ordem. Se algum passo contradisser o `.r.spec.md`, **pare** — o `.r` tem precedência.
> Todo código novo nasce de teste que falhou primeiro (`docs/conventions/TESTING.pt-BR.md`).

## Duas etapas, dois marcos

Este componente entrega em dois momentos do roteiro:

| Etapa | Passos | Marco | Entrega |
|---|---|---|---|
| **A — Política** | 1 a 3 | Fase 4 | Avaliador puro; o loop pode chamá-lo desde o primeiro executável |
| **B — Fronteira** | 4 a 7 | Fase 6 | Sandbox do SO ligado; a tese do produto passa a valer |

Separar assim é o que permite o loop cumprir RN-7 antes de o sandbox existir, sem criar caminho alternativo depois.

---

## Etapa A — Política

### Passo 1 — Tipos e tabela de decisão

`internal/policy/policy.go`

- [ ] `SandboxMode`, `ApprovalPolicy`, `Request`, `Access`, `Decision`, `Verdict`.
- [ ] Constantes com os nomes exatos do `.p` — são `stable`, aparecem em config de usuário.

**Teste obrigatório:** valor desconhecido de modo ou de política é erro explícito, nunca cai em default silencioso.

### Passo 2 — `Evaluate`, puro

`internal/policy/policy.go`

- [ ] Tabela de modo (15 células) e filtro de política (12 células) da seção 3.1 do `.p`.
- [ ] Sem I/O, sem relógio.

**Testes obrigatórios:**
- Uma asserção por célula das duas tabelas — 27 no total, table-driven.
- `read-only` nunca devolve `allow` para escrita, sob nenhuma política.
- `never` nunca devolve `escalate`.
- Guarda de pureza: o pacote não importa `os`, `net` nem `time`.

> Tabela completa em vez de amostragem porque cada célula é uma decisão de segurança, e a que ninguém testou é exatamente a que vai estar errada.

### Passo 3 — `Resolve`

`internal/policy/resolve.go`

- [ ] Relativo resolve contra o workspace, nunca contra o diretório do processo.
- [ ] Symlink resolvido até o alvo final.
- [ ] `..` normalizado antes de comparar.
- [ ] Caminho inexistente resolve o pai mais próximo existente.
- [ ] Contenção por componente, nunca por prefixo de string.

**Testes obrigatórios, com `t.TempDir()` e arquivos reais:**
- Symlink dentro do workspace apontando para fora **é** cruzamento.
- `/home/user/proj2` não é contido em `/home/user/proj`.
- `../../etc/passwd` é cruzamento, não erro.
- Arquivo ainda inexistente dentro do workspace é permitido para escrita.

> Sem mock aqui. Mockar `filepath` e `os.Readlink` testaria o mock — e é justamente a semântica real do sistema de arquivos que estamos verificando (`TESTING.pt-BR.md`, exceção deliberada do TDD London).

### 🎯 Fim da Etapa A — o loop pode cumprir RN-7

---

## Etapa B — Fronteira do sistema operacional

### Passo 4 — Interface e detecção

`internal/sandbox/sandbox.go`

- [ ] Interface `Sandbox` com `Wrap` e `Available`.
- [ ] Seleção por `DCODE_SANDBOX_BACKEND`, com `auto` decidindo pelo SO.
- [ ] `none` aceito **apenas** com `full-access`; qualquer outra combinação é erro de inicialização.

**Teste obrigatório:** `Available()` falhando impede a criação da sessão. Nenhum caminho de código executa comando sem fronteira estabelecida (RN-3).

### Passo 5 — Backend macOS

`internal/sandbox/seatbelt/`

- [ ] Geração de perfil `sandbox-exec` a partir de workspace e modo.
- [ ] Execução por `exec` do binário — **sem cgo**.
- [ ] Perfil escrito em `DCODE_SANDBOX_PROFILE_DIR`.

**Testes obrigatórios, executando de verdade:**
- Em `read-only`, escrita no workspace falha pelo SO, não por verificação em Go.
- Em `workspace-write`, escrita fora do workspace falha pelo SO.
- Com rede negada, conexão de saída falha pelo SO.
- Golden file do perfil gerado, por modo.

> A asserção que importa é "falhou **pelo sistema operacional**". Teste que só verifica o retorno em Go passaria mesmo com o sandbox desligado — e é exatamente o teste inútil que dá falsa confiança.

### Passo 6 — Backend Linux

`internal/sandbox/bubblewrap/`

- [ ] Mesmos testes do Passo 5, com `bwrap`.
- [ ] Namespaces de usuário não-privilegiados; sem cgo.
- [ ] Erro legível quando a distribuição restringe namespaces — nomear o ajuste necessário, não só falhar.

> Ubuntu 24.04 e posteriores restringem namespaces não-privilegiados por AppArmor. É a causa mais provável de falha de instalação no Linux e merece a mensagem mais cuidadosa do projeto.

### Passo 7 — Governança e invariantes

- [ ] Leitura de `DCODE_REQUIREMENTS_FILE`; valores travados vencem ambiente e flag (RN-7).
- [ ] Um teste por linha da seção 6 do `.p.spec.md`.
- [ ] Matriz de CI cobrindo macOS e Linux; pacote de backend fora da plataforma sai do denominador de cobertura.
- [ ] `go test -race ./internal/policy/... ./internal/sandbox/...` limpo.

## Ordem de dependência

```
Passo 1 (tipos)
  └─ Passo 2 (Evaluate, puro)
       └─ Passo 3 (Resolve)         ← 🎯 fim da Etapa A
            └─ Passo 4 (interface)
                 ├─ Passo 5 (macOS)
                 └─ Passo 6 (Linux)
                      └─ Passo 7 (governança e invariantes)
```

## Armadilhas conhecidas

- **Contenção por prefixo de string** — `strings.HasPrefix(path, workspace)` deixa `/proj2` passar por `/proj`. É o bug de fronteira mais comum que existe.
- **Symlink verificado antes de resolver** — inverte a ordem e a fronteira cai com um `ln -s`.
- **Teste que verifica o retorno em Go, não o efeito no SO** — passa com o sandbox desligado. Sempre asserte o efeito real.
- **cgo entrando pelo backend** — um `import "C"` aqui quebra `CGO_ENABLED=0` e a compilação cruzada. Todos os backends são `exec`.
- **Degradar quando o binário falta** — o caminho tentador é "avisa e segue sem sandbox". Viola RN-3 e é pior que não ter sandbox, porque promete o que não entrega.
- **Perfil gerado com caminho não resolvido** — o perfil precisa do workspace já canonicalizado, senão a fronteira do SO difere da avaliada em Go.

## Changelog

_Sem alterações desde a criação._
