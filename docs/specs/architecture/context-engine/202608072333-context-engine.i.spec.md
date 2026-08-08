# Implementing: Motor de Contexto

> Siga a ordem. Se algum passo contradisser o `.r.spec.md`, **pare** — o `.r` tem precedência.
> Todo código novo nasce de teste que falhou primeiro (`docs/conventions/TESTING.pt-BR.md`).

## Ordem de execução

### Passo 1 — Tipos, sem lógica

`internal/context/types.go`

- [ ] `Role` e constantes; `Message`, `ToolCall`, `ToolResult`, `ToolDef`, `Summary`, `Session`.
- [ ] Testes de ida e volta de JSON com golden em `testdata/`.

> Sem I/O, sem dependência de outro pacote do projeto além de `encoding/json`.

### Passo 2 — `Estimate`

`internal/context/estimate.go`

- [ ] Heurística por caracteres com razão e margem do `.config`.
- [ ] Determinística: mesma entrada, mesmo valor.

**Teste obrigatório:** 1000 chamadas com a mesma entrada retornam valor idêntico.

> Vem antes de `Assemble` porque a compactação depende dela e ela não depende de nada.

### Passo 3 — `Assemble`

`internal/context/assemble.go`

- [ ] Ordem de seções exatamente conforme tabela 4 do `.p`.
- [ ] `Summary == nil` omite a seção por completo, sem marcador.
- [ ] Nenhum I/O, nenhum relógio, nenhuma leitura de ambiente.

**Testes obrigatórios:**
- Idempotência byte-a-byte para a mesma `Session`.
- **Estabilidade de prefixo:** monta, anexa mensagem, monta de novo — o prefixo comum é idêntico byte a byte. Este é o teste que guarda a ADR-03 inteira; se ele quebrar, o produto perdeu sua principal vantagem de custo.
- Ausência de token volátil na saída, varrendo por padrão de timestamp e por ID de sessão.
- Golden file cobrindo todas as combinações de campo presente e ausente.

### Passo 4 — Guarda de pureza

`internal/context/purity_test.go`

- [ ] Teste que falha se o pacote importar `os`, `net`, `time` (fora de tipo), `math/rand` ou `syscall`.

> Parece exagero até o dia em que alguém adiciona um `time.Now()` "só para log" e o cache cai em produção sem ninguém entender por quê. O teste é mais barato que a investigação.

### Passo 5 — `Plan`

`internal/context/compaction.go`

- [ ] Gatilho por `CompactAt` × janela.
- [ ] `ToIdx` sempre em fronteira de turno completo.
- [ ] Última `RoleUser` e posteriores nunca no trecho compactado (RN-6).
- [ ] `KeepTurns` do `.config` respeitado.

**Testes obrigatórios:**
- Nenhum plano separa `RoleAssistant` com `ToolCalls` dos seus `RoleTool` — varrer com histórico gerado por tabela, incluindo tool call no limite exato do corte.
- Nenhum plano inclui a última `RoleUser`.
- Abaixo do gatilho, `Plan` retorna `false` e não altera nada.

### Passo 6 — Invariantes

`internal/context/invariants_test.go`

- [ ] Um teste por linha da seção 7 do `.p.spec.md`.
- [ ] `go test -race ./internal/context/...` limpo.
- [ ] Cobertura ≥ 90% neste pacote.

## Ordem de dependência

```
Passo 1 (tipos)
  ├─ Passo 2 (Estimate)
  │    └─ Passo 5 (Plan)
  └─ Passo 3 (Assemble)
       ├─ Passo 4 (guarda de pureza)
       └─ Passo 6 (invariantes)
```

## Fora deste componente

- **Gerar o texto do resumo** chama o modelo — pertence ao loop do agente. Aqui só o plano.
- **Traduzir `[]Message` para o formato de fio do provedor** pertence ao adaptador de provider.
- **Decidir quando chamar o modelo** pertence ao loop.

Manter essas três coisas fora é o que preserva a pureza, e a pureza é o que torna o componente exatamente testável.

## Armadilhas conhecidas

- **`map` em serialização** — ordem de iteração de map em Go é aleatória por design. Qualquer estrutura que vire JSON precisa de campo ordenado ou de `json.Marshal` sobre struct, nunca sobre `map` com múltiplas chaves.
- **`time.Time` em tipo do prefixo** — mesmo não serializado, convida ao uso. Não existe campo de tempo em nada que entre no contexto.
- **Corte no meio de tool call** — o caso que mais escapa é a compactação que corta entre o `RoleAssistant` e o último dos seus `RoleTool`. Teste com o corte caindo exatamente ali.
- **`Estimate` mudando de valor entre versões** — quebra golden de compactação. Mudança na heurística é mudança de comportamento e exige regenerar golden conscientemente, não por acidente.

## Changelog

_Sem alterações desde a criação._
