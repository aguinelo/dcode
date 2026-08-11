# Implementing: Adaptador de Provider

> Siga a ordem. Se algum passo contradisser o `.r.spec.md`, **pare** — o `.r` tem precedência.
> Todo código novo nasce de teste que falhou primeiro (`docs/conventions/TESTING.pt-BR.md`).

## Pré-requisito

O motor de contexto (`202608072333-context-engine`) precisa estar no Passo 3 — os tipos `Message`, `ToolCall`, `ToolDef` são a fronteira neutra deste componente.

## Ordem de execução

### Passo 1 — Os dois eixos, sem implementação

`internal/provider/provider.go`

- [ ] `Transport` e `Family` como interfaces separadas.
- [ ] `Provider` como composição; `Limits`, `Request`, `WireRequest`, `WireEvent`.
- [ ] `StreamEvent`, `StreamEventType`, `Usage`.
- [ ] `ProviderError`, `ErrorClass` e constantes.
- [ ] `Registry` com `RegisterTransport`, `RegisterFamily` e `Resolve`.

**Testes obrigatórios:**
- Modelo desconhecido devolve erro listando as famílias disponíveis; nunca resolve para família default.
- Prefixos de `Models()` sobrepostos entre famílias falham na inicialização.
- Transporte fora de `Transports()` da família devolve erro nomeando os compatíveis.
- `Limits()` devolve o default da família quando a config não sobrescreve.

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

### Passo 4 — Transporte `openai`

`internal/provider/transport/openai/`

- [ ] `Do` sobre HTTP com SSE; sem qualquer conhecimento de família.
- [ ] Parsing do envelope bruto em `WireEvent`, contra transcript gravado.

> O transporte não conhece prompt, schema de tool nem limiar. Se precisar de `if familia == X`, os eixos foram misturados.

### Passo 5 — Família `minimax-m3`

`internal/provider/family/minimaxm3/`

**Primeira família implementada — é o modelo principal do projeto.**

- [ ] `Models()`, `Transports()` devolvendo `["openai", "anthropic"]`, `Window`.
- [ ] `DefaultLimits()` com `MaxIterations: 200`.
- [ ] `Encode` para o transporte `openai`, com golden file.
- [ ] `Decode` de `WireEvent` em `StreamEvent`.
- [ ] Validação de tool call contra o schema declarado; falha vira `ErrClassToolSchema`, nunca `EventToolCall` (RN-8).
- [ ] Filtro de nome de ferramenta fora do conjunto declarado.

**Teste obrigatório:** `Usage.CacheReadTokens` preenchido quando o provedor informa — é a medida direta de que a ADR-03 está funcionando.

### Passo 6 — Transporte `anthropic` e família `claude`

`internal/provider/transport/anthropic/`, `internal/provider/family/claude/`

- [ ] Transporte `anthropic`, mesmos testes do Passo 4.
- [ ] Família `claude` com `Transports()` devolvendo `["anthropic"]` e `MaxIterations: 50`.
- [ ] `Encode` do `minimax-m3` para o transporte `anthropic`.

**O teste que justifica a arquitetura inteira:** a família `minimax-m3` codificando para `openai` e para `anthropic` produz corpos **distintos e ambos válidos**, sem duplicar adaptação nem limiar. Se este teste for difícil de escrever, os eixos não estão ortogonais de fato.

**Testes obrigatórios:**
- Todo stream termina em exatamente um `EventDone` ou um `EventError`.
- `ctx` cancelado fecha o canal com `ErrClassCanceled`.
- Tool call malformada não chega ao consumidor.
- `Usage.CacheReadTokens` é preenchido quando o provedor informa — é a medida direta de que a ADR-03 está funcionando.

### Passo 7 — Classificação de erro e retry

`internal/provider/retry.go`

- [ ] Mapeamento de cada erro do provedor para `ErrorClass`, com transcript de erro gravado por classe.
- [x] Recuo exponencial com `RetryBaseDelay` e teto `RetryMaxDelay`.
- [ ] `rate_limit` usa `RetryAfter` do provedor, ignorando o recuo.
- [ ] `auth`, `quota` e `bad_request` nunca repetem.

**Teste obrigatório:** um transcript por classe da tabela da seção 4 do `.p`, verificando a decisão de retry de cada uma.

### Passo 8 — Guarda de credencial

`internal/provider/secret_test.go`

- [ ] Injeta chave sentinela em `DCODE_API_KEY`, exercita erro de cada classe, e varre `ProviderError.Message`, log e evento em busca da sentinela (RN-6).

**Falha neste teste é blocker imediato.** Credencial em log é o tipo de vazamento que só se descobre depois de publicado.

### Passo 9 — Contratos comportamentais

`internal/evals/` — a parte que alcança modelo atrás de build tag `eval`.

> **Era `internal/provider/evals/`.** Não pode ser: o cenário precisa de um
> transporte de verdade, e o transporte HTTP vive em `internal/app`, que importa
> `internal/provider`. Um pacote de eval sob `provider` fecharia o ciclo. Como
> `internal/evals` é folha — ninguém a importa —, ela pode importar as duas, e é
> onde os contratos das demais specs também vão morar.

- [x] Fixture para `toolcall-schema-valid`, `toolcall-recover` e `no-phantom-tool`.
- [x] Executor que roda `DCODE_EVAL_RUNS` vezes e compara com o limiar. Um executor para **todos** os contratos, nao um por spec: `Contracts` liga cada ID ao seu juizo, e duas guardas impedem fixture sem juizo e juizo sem fixture.
- [x] Registro do modelo e versão medidos junto do resultado.
- [x] Não roda na suíte padrão nem na CI de PR.
- [x] `Measure` fica **fora** da build tag. Ela recebe a tentativa como função e
      não alcança modelo nenhum, então é testável na suíte normal — e decidir se
      um limiar foi cumprido é exatamente o que não pode depender de medição.
- [x] Erro de transporte não é veredito. Uma execução que falhou em medir conta
      em `Errors`, e resultado com erro **nunca** é `Met`: queda de rede lida
      como regressão de comportamento é o erro que este pacote existe para
      impedir.
- [x] `make eval` roda; `make eval-build` só compila, e é o que impede a suíte
      apodrecer em silêncio enquanto o código que ela mede muda por baixo.

> Estes são os primeiros contratos comportamentais do projeto. Se um limiar não for atingível, o achado é sobre a família de modelo, não sobre o código — e a conclusão pode ser rebaixar o limiar **com** entrada em `changelog/`, ou não suportar aquela família.

### Passo 10 — Invariantes

`internal/provider/invariants_test.go`

- [ ] Um teste por linha da seção 7 do `.p.spec.md`.
- [ ] `go test -race ./internal/provider/...` limpo.
- [ ] Cobertura ≥ 90%, excluído o pacote de eval (build tag).

## Ordem de dependência

```
Passo 1 (dois eixos)
  ├─ Passo 2 (guarda de fronteira)
  └─ Passo 3 (transcript)          ← antes de qualquer adaptador real
       └─ Passo 4 (transporte openai)
            └─ Passo 5 (família minimax-m3)   ← 🎯 modelo principal, primeiro
                 └─ Passo 6 (transporte anthropic + família claude)
                      ├─ Passo 7 (erro e retry)
                      ├─ Passo 8 (guarda de credencial)
                      ├─ Passo 9 (contratos comportamentais)
                      └─ Passo 10 (invariantes)
```

> M3 vem primeiro por ser o modelo principal do projeto: é contra ele que os limiares são medidos primeiro. Claude entra no Passo 6 porque é o que **prova a abstração** — uma implementação nunca valida um eixo ortogonal.

## Fora deste componente

- **Decidir repetir ou desistir** é do loop; aqui só a classificação que permite a decisão.
- **Montar o contexto** é do motor de contexto; aqui só a tradução para o fio.
- **Executar ferramenta** é do loop e do sandbox.

## Armadilhas conhecidas

- **Canal que não fecha** — stream que termina sem `EventDone` nem `EventError` pendura o loop para sempre. Cobrir com transcript truncado no meio.
- **Vazamento de tipo de SDK** — um `*sdk.Message` num campo exportado destrói a ADR-05 silenciosamente. É o que o Passo 2 impede.
- **Transporte conhecendo família** — um `if familia == X` dentro do transporte colapsa os dois eixos de volta em um, e o sintoma só aparece na terceira família.
- **Família assumindo um transporte só** — `Encode` que ignora o parâmetro de transporte funciona enquanto houver um dialeto e quebra em silêncio no segundo.
- **Credencial em `%v`** — um `fmt.Errorf("request failed: %v", req)` com a chave dentro da struct vaza sem ninguém notar. O Passo 6 existe por isso.
- **Retry em erro não idempotente** — repetir uma chamada que já teve efeito no lado do provedor duplica cobrança. Só repetir o que falhou antes de ser aceito.
- **Transcript gravado com credencial dentro** — sanitizar no gravador, não na revisão. Transcript vai para `testdata/`, que é versionado e público.

## Changelog

- [202608072352 — Transporte e família como eixos ortogonais](changelog/202608072352-transporte-familia-ortogonais.md)
