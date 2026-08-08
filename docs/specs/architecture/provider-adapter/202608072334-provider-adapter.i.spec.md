# Implementing: Adaptador de Provider

> Siga a ordem. Se algum passo contradisser o `.r.spec.md`, **pare** — o `.r` tem precedência.
> Todo código novo nasce de teste que falhou primeiro (`docs/conventions/TESTING.pt-BR.md`).

## Pré-requisito

O motor de contexto (`202608072333-context-engine`) precisa estar no Passo 3 — os tipos `Message`, `ToolCall`, `ToolDef` são a fronteira neutra deste componente.

## Ordem de execução

### Passo 1 — Interface e tipos, sem implementação

`internal/provider/provider.go`

- [ ] `Provider`, `Request`, `StreamEvent`, `StreamEventType`, `Usage`.
- [ ] `ProviderError`, `ErrorClass` e constantes.
- [ ] `Registry` com `Register` e `Resolve`.

**Teste obrigatório:** modelo desconhecido devolve erro listando os prefixos suportados; nunca resolve para adaptador default.

> Nenhuma chamada de rede neste passo. A interface existe antes da primeira implementação justamente para que a segunda família não force reescrita (ADR-05).

### Passo 2 — Guarda de fronteira

`internal/provider/boundary_test.go`

- [ ] Teste que falha se qualquer tipo específico de provedor for exportado do pacote (RN-2).
- [ ] Teste que falha se pacote fora de `internal/provider` importar SDK de provedor.

> Vem cedo de propósito. A ADR-05 se perde por vazamento gradual, não por decisão — e vazamento só é barato de impedir antes de existir.

### Passo 3 — Reprodutor de transcript

`internal/provider/transcript/`

- [ ] Formato de gravação de stream em `testdata/transcripts/`.
- [ ] Reprodutor que implementa `Provider` a partir de arquivo gravado.
- [ ] Gravador atrás de build tag, para capturar transcript novo de rede real.

**Teste obrigatório:** reproduzir o mesmo transcript duas vezes produz sequência idêntica de `StreamEvent`.

> Este passo vem **antes** do primeiro adaptador real. É o que torna todo o resto testável sem rede (RN-4) e é a base do gate de cobertura. Construir o adaptador primeiro e o reprodutor depois inverte a ordem e gera código não testado.

### Passo 4 — Primeiro adaptador de família

`internal/provider/<família>/`

- [ ] Tradução de `[]Message` para o formato de fio, com golden file.
- [ ] Tradução de `[]ToolDef` para o schema de ferramenta da família.
- [ ] Parsing do stream em `StreamEvent`, contra transcript gravado.
- [ ] `Window` por modelo conhecido.
- [ ] Validação de tool call contra o schema declarado; falha vira `ErrClassToolSchema`, nunca `EventToolCall` (RN-8).
- [ ] Filtro de nome de ferramenta fora do conjunto declarado.

**Testes obrigatórios:**
- Todo stream termina em exatamente um `EventDone` ou um `EventError`.
- `ctx` cancelado fecha o canal com `ErrClassCanceled`.
- Tool call malformada não chega ao consumidor.
- `Usage.CacheReadTokens` é preenchido quando o provedor informa — é a medida direta de que a ADR-03 está funcionando.

### Passo 5 — Classificação de erro e retry

`internal/provider/retry.go`

- [ ] Mapeamento de cada erro do provedor para `ErrorClass`, com transcript de erro gravado por classe.
- [ ] Recuo exponencial com `RetryBaseDelay` e teto `RetryMaxDelay`.
- [ ] `rate_limit` usa `RetryAfter` do provedor, ignorando o recuo.
- [ ] `auth`, `quota` e `bad_request` nunca repetem.

**Teste obrigatório:** um transcript por classe da tabela da seção 4 do `.p`, verificando a decisão de retry de cada uma.

### Passo 6 — Guarda de credencial

`internal/provider/secret_test.go`

- [ ] Injeta chave sentinela em `DCODE_API_KEY`, exercita erro de cada classe, e varre `ProviderError.Message`, log e evento em busca da sentinela (RN-6).

**Falha neste teste é blocker imediato.** Credencial em log é o tipo de vazamento que só se descobre depois de publicado.

### Passo 7 — Contratos comportamentais

`internal/provider/evals/` — atrás de build tag `eval`.

- [ ] Fixture para `toolcall-schema-valid`, `toolcall-recover` e `no-phantom-tool`.
- [ ] Executor que roda `DCODE_EVAL_RUNS` vezes e compara com o limiar.
- [ ] Registro do modelo e versão medidos junto do resultado.
- [ ] Não roda na suíte padrão nem na CI de PR.

> Estes são os primeiros contratos comportamentais do projeto. Se um limiar não for atingível, o achado é sobre a família de modelo, não sobre o código — e a conclusão pode ser rebaixar o limiar **com** entrada em `changelog/`, ou não suportar aquela família.

### Passo 8 — Invariantes

`internal/provider/invariants_test.go`

- [ ] Um teste por linha da seção 7 do `.p.spec.md`.
- [ ] `go test -race ./internal/provider/...` limpo.
- [ ] Cobertura ≥ 90%, excluído o pacote de eval (build tag).

## Ordem de dependência

```
Passo 1 (interface)
  ├─ Passo 2 (guarda de fronteira)
  └─ Passo 3 (transcript)      ← antes do adaptador real, não depois
       └─ Passo 4 (família)
            ├─ Passo 5 (erro e retry)
            ├─ Passo 6 (guarda de credencial)
            ├─ Passo 7 (contratos comportamentais)
            └─ Passo 8 (invariantes)
```

## Fora deste componente

- **Decidir repetir ou desistir** é do loop; aqui só a classificação que permite a decisão.
- **Montar o contexto** é do motor de contexto; aqui só a tradução para o fio.
- **Executar ferramenta** é do loop e do sandbox.

## Armadilhas conhecidas

- **Canal que não fecha** — stream que termina sem `EventDone` nem `EventError` pendura o loop para sempre. Cobrir com transcript truncado no meio.
- **Vazamento de tipo de SDK** — um `*sdk.Message` num campo exportado destrói a ADR-05 silenciosamente. É o que o Passo 2 impede.
- **Credencial em `%v`** — um `fmt.Errorf("request failed: %v", req)` com a chave dentro da struct vaza sem ninguém notar. O Passo 6 existe por isso.
- **Retry em erro não idempotente** — repetir uma chamada que já teve efeito no lado do provedor duplica cobrança. Só repetir o que falhou antes de ser aceito.
- **Transcript gravado com credencial dentro** — sanitizar no gravador, não na revisão. Transcript vai para `testdata/`, que é versionado e público.

## Changelog

_Sem alterações desde a criação._
