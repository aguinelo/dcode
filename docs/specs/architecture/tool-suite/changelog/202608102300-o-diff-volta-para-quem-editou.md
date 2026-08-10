# O diff volta para quem editou

**Data:** 2026-08-10
**Specs afetadas:** `202608072337-tool-suite` (`.r`, `.p`, `.config`)

> **Regra:** o diff volta ao modelo exatamente quando ele **não consegue derivar** o resultado — e só então.

## O problema medido

O diff é calculado em toda escrita e toda edição — `internal/tools/file.go:190` e `:292`. O comentário no tipo diz para onde ele vai (`internal/tools/tool.go:56`):

```go
// Diff is the unified diff of a change, for a client to render.
```

**Para o cliente desenhar.** Vai para a tela. O que volta ao modelo é `"edited internal/foo.go (3 replacement(s), +7 −4)"` — uma contagem.

O modelo — o único que pode corrigir no mesmo turno — é o único que não vê.

## O agravante: a sessão marca como lido o que ninguém leu

`file.go:283`, logo após a escrita:

```go
s.MarkRead(abs, updated, 0)
```

A linha é **necessária**: sem ela a segunda edição do mesmo arquivo falharia como `file_changed`, e o comentário no código já explica isso.

Mas a consequência é que a invariante read-before-edit da RN-3 passa a ser satisfeita por uma leitura **que o modelo não fez**. Formalmente ele está em dia com o conteúdo; de fato só o harness viu aquele conteúdo. Numa cadeia de edições, a distância entre o que o modelo acha que está no arquivo e o que está lá cresce a cada passo, sem nada sinalizar.

**Devolver o diff é o que torna essa marcação honesta** — e é por isso que a regra abaixo casa exatamente com o caso em que ela mente.

## Por que não é "devolve sempre"

O contexto é append-only (ADR-03). O diff entra no histórico e é pago **em todo turno seguinte, para sempre**. Vinte edições num refactor, com teto de 400 linhas por diff, é histórico que estoura sozinho — e o Tema do orçamento de contexto acabaria consumido por isto.

Então a pergunta não é *devolver ou não*, é **quando**.

## O critério: o modelo consegue derivar o resultado?

| Caso | O modelo sabe o que ficou no arquivo? | Diff volta? |
|---|---|---|
| `write` | **sim** — ditou cada byte do conteúdo | não |
| `write` sobre existente | sim — a RN-8 já exige leitura prévia | não |
| `edit`, uma ocorrência | **sim** — `old_string` é único, logo o local é determinado | não |
| `edit` com `replace_all`, N > 1 | **não** — ele não sabe onde estavam as N ocorrências | **sim** |

A linha é nítida porque a RN-4 já fez metade do trabalho: match ambíguo **falha**, nunca adivinha. Sobra um único caso em que a edição acontece sem o modelo saber onde — `replace_all` deliberado sobre múltiplas ocorrências.

É exatamente onde mora "substituí no lugar errado", e é exatamente onde `MarkRead` marca como visto um conteúdo que ninguém olhou. Um caso, não uma política geral: **custo praticamente zero no uso comum.**

## Truncamento

O diff devolvido respeita `DiffMaxLines` e **declara** o corte, como a RN-5 já exige de toda saída. `UnifiedDiff` já emite `⋯ diff truncated at N lines`; nenhuma mudança ali.

Diff truncado em silêncio seria o pior dos mundos: o modelo concluiria sobre a parte que não viu, com a confiança de quem acha que viu tudo.

## Configuração

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_EDIT_ECHO_DIFF` | enum | `multi` | `never` devolve só a contagem, como hoje. `multi` devolve o diff apenas em `replace_all` que afetou mais de uma ocorrência. `always` devolve em toda edição — custo alto no histórico, útil para depurar um modelo que está editando errado. |

`write` não tem modo: nunca devolve diff, em nenhum valor da chave. Não é economia — é que a informação seria literalmente aquilo que o modelo acabou de ditar.

## Fronteira de determinismo

| Parte | Regime | Verificação |
|---|---|---|
| quando o diff volta | determinístico | asserção |
| conteúdo do diff | determinístico | golden |
| declaração de truncamento | determinístico | asserção |
| **agir sobre o diff devolvido** | **mediado** | limiar |

## Contratos comportamentais

| ID | Cenário | Comportamento esperado | Limiar |
|---|---|---|---|
| `notices-wrong-replacement` | `replace_all` acerta uma ocorrência indevida, visível no diff | percebe e corrige no mesmo turno | ≥ 85% |

Um contrato só, porque o resto é determinístico.

## Relação com a verificação

Não se substituem. `202608102000` pergunta **"você rodou?"**; esta pergunta **"você olhou?"**.

Passar nos testes não prova que o diff é o pretendido — prova que o que sobrou compila e passa. Função duplicada, comentário destruído, linha certa apagada num trecho sem cobertura: tudo isso passa verde.

## Impacto

- `Result.Output` de `edit` passa a incluir o diff no caso de `replace_all` múltiplo; `Meta.Diff` continua indo ao cliente em todos os casos, como hoje.
- Nenhuma mudança em `write`, em `UnifiedDiff` ou no cálculo.
- Uma chave nova; nenhuma constante deixa de ser constante.
