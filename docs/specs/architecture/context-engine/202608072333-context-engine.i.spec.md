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

### Passo N — Orçamento realimentado

> Acrescentado por [202608102200](changelog/202608102200-orcamento-de-contexto-realimentado.md). Este arquivo dizia "sem alterações desde a criação" enquanto o `.r`, o `.p` e o `.config` já carregavam a RN-6.1, três tipos novos e seis invariantes. Guia de implementação que não acompanha a regra é guia que ninguém segue.

`internal/contextengine/budget.go`

- [x] `Fraction(s Session, cfg Config) float64` — a mesma conta que `Plan` já faz, agora alcançável por quem pode agir sobre ela. Pura.
- [x] `Band`, `BandNone`/`Band60`/`Band80`/`Band92`, e `BandFor(f, compactAt float64) Band`.
- [x] `Crossed(announced Band, f, compactAt float64) (Band, bool)` — só para cima. Descer não anuncia, mas **rearma**.
- [x] Epsilon na comparação de limiar. `Fraction` vem de heurística de caractere com margem: diferença de `1e-16` não carrega informação, e sem o epsilon um valor exatamente no limiar cai para a faixa de baixo, porque `0.64/0.80` é `0.7999…` em binário.

`internal/behavior/reminders.go`

- [x] `ReminderContextBudget`, **um** `Kind` com três textos escolhidos pela faixa. A regra é uma; o que muda é quanto resta.
- [x] Texto **constante** por faixa. Nenhum número interpolado: valor que muda a cada turno torna o histórico irreproduzível (RN-7).
- [x] `SessionState.BudgetCrossed`, preenchido só na travessia. Quem decide se houve travessia é o chamador, porque isso exige a faixa anterior e `Emit` é função só deste estado.

`internal/loop/turn.go`

- [x] A faixa já anunciada vive no `Engine`, não é derivada por turno — é o que torna a emissão por borda em vez de por nível.
- [x] `crossBudget()` roda uma vez por iteração, junto da checagem de compactação que já recalcula a fração.

**Testes obrigatórios:**
- `BandFor` e `Crossed` puros, com o limiar exato incluído.
- Anuncia uma vez; permanecer na mesma faixa não anuncia de novo.
- Descer não anuncia **e** rearma: subir de novo anuncia.
- Sessão curta não anuncia nada — a metade determinística de `no-budget-noise-when-low`.
- Nenhuma faixa cai em cima ou acima de `CompactAt`, para qualquer gatilho configurado.

> **A correção que a implementação forçou.** As faixas são fração do **orçamento**, não da janela. Ver a nota no `.config`: `0.80` e `0.92` da janela são inalcançáveis com `CompactAt` em `0.80`, e o teste que exige toda faixa abaixo do gatilho é o que pegou.

## Armadilhas conhecidas

- **`map` em serialização** — ordem de iteração de map em Go é aleatória por design. Qualquer estrutura que vire JSON precisa de campo ordenado ou de `json.Marshal` sobre struct, nunca sobre `map` com múltiplas chaves.
- **`time.Time` em tipo do prefixo** — mesmo não serializado, convida ao uso. Não existe campo de tempo em nada que entre no contexto.
- **Corte no meio de tool call** — o caso que mais escapa é a compactação que corta entre o `RoleAssistant` e o último dos seus `RoleTool`. Teste com o corte caindo exatamente ali.
- **`Estimate` mudando de valor entre versões** — quebra golden de compactação. Mudança na heurística é mudança de comportamento e exige regenerar golden conscientemente, não por acidente.

## Changelog

- [202608102200 — Orçamento de contexto realimentado](changelog/202608102200-orcamento-de-contexto-realimentado.md)
