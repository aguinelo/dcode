# Busca por símbolo

**Data:** 2026-08-10
**Specs afetadas:** `202608072337-tool-suite` (`.r`, `.p`, `.config`)

> **Regra:** procurar uma coisa é diferente de procurar as letras dela, e o conhecimento de como uma linguagem declara algo pertence à ferramenta, não ao modelo.

## O problema

O conjunto tem `glob` (nome de arquivo) e `grep` (texto). Pedindo *"renomeie `Parse` para `Decode`"*, o modelo roda `grep Parse` e recebe:

- `Parse`, a função procurada ✓
- `ParseTOML`, `ParseInterval`, `parseMode` — nomes distintos com as mesmas letras
- `// Parse é aproximada` — comentário
- `"failed to parse"` — dentro de string
- `json.Parse` — outra biblioteca

Cada falso positivo custa um arquivo lido para descartar, e arquivo lido é contexto — o mesmo orçamento de `202608102200`. **Busca imprecisa é o que enche a memória**; os dois problemas são um só, visto de dois ângulos.

## O que este changelog corrige e o que não corrige

Um levantamento honesto muda o desenho: **`grep` aceita expressão regular Go**, então `\bParse\b` já é expressável hoje. A capacidade existe.

O que falta não é poder — é **o que a ferramenta induz**. Nada faz da fronteira de símbolo o caminho natural, e o modelo escreve `Parse`. Duas consequências:

- Parte da correção é **descrição de ferramenta**, que a RN-1 de `behavior-definition` classifica como a camada de alta precisão, lida no momento da decisão.
- Descrição sozinha é a camada fraca, esquecida em conversa longa. Precisa de estrutura junto.

## A oitava ferramenta, e o que a justifica

A RN-1 desta spec fixa um conjunto mínimo, e acrescentar exige justificativa. `symbol` a tem, e **não** é a fronteira de palavra — isso é uma regex que o modelo poderia escrever.

É a **distinção entre definição e uso**. Para expressá-la em `grep`, o modelo precisaria saber como cada linguagem declara: `func Parse(` em Go, `def parse` em Python, `fn parse` em Rust, `function`/`const … =` em TypeScript. Isso é conhecimento que **a ferramenta carrega como dado** — uma tabela de padrões por extensão — em vez de o modelo reconstruir por linguagem, toda vez, e errar em metade delas.

```go
type SymbolInput struct {
    Name string `json:"name"`           // o símbolo, sem regex
    Kind string `json:"kind,omitempty"` // "def" | "ref" | "any"; default "any"
    Path string `json:"path,omitempty"`
    Glob string `json:"glob,omitempty"`
}
```

`Name` **não é expressão regular**: é escapado antes de virar padrão. Aceitar regex aqui reintroduziria o problema pela porta dos fundos e faria `symbol` ser `grep` com outro nome.

O resultado é `path:line:text` com contexto, ordenado — nunca o arquivo inteiro. É o que evita ler tudo para descobrir que não era.

## O que ela não resolve, e por que isso precisa estar no resultado

`symbol` é textual. **Não resolve despacho dinâmico** — quem chama por interface, por ponteiro de função, por reflexão ou por nome montado em tempo de execução não aparece.

Isso é falso negativo, que é o pior modo de falha possível: **resultado vazio parecendo resposta completa**. Um agente que conclui "não há outros chamadores" de uma busca textual renomeia metade das coisas com confiança.

Por isso o resultado **declara o próprio limite**, sempre:

```
symbol: Parse (def) — 1 match, 12 refs
textual match on symbol boundary; does not resolve interface or dynamic dispatch
```

É o mesmo princípio da RN-5, que já exige que truncamento seja declarado, e pela mesma razão: saída não pode parecer completa quando não é.

Extensão sem padrão conhecido **não é erro** — cai na busca por fronteira de palavra, com `kind` reportado como desconhecido. Recusar a linguagem seria pior que responder com o limite declarado.

## O passo seguinte, que continua fora

O `.r` já registrava, e a decisão continua de pé:

> *"Consulta a language server. Reconhecido como diferencial real do opencode, mas não é o mínimo."*

Só o language server resolve o falso negativo do despacho dinâmico. O custo é um servidor por linguagem, cada um com instalação, ciclo de vida e protocolo próprios.

`symbol` **não** é um substituto: é o degrau que corta o custo do caminho comum, que é onde o contexto está sendo gasto hoje. Quando o language server entrar, `symbol` passa a ser o fallback para linguagem sem servidor — e a declaração de limite no resultado é o que permite a troca sem mudar como o modelo interpreta a saída.

## Configuração

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_SYMBOL_MAX_MATCHES` | inteiro | `200` | Mesmo teto de `grep`, e pelo mesmo motivo: símbolo que casa milhares de vezes é símbolo mal escolhido, e devolver tudo gasta contexto sem informar. |

## Invariantes

- `Name` é escapado; `symbol` com `Name = "a.b"` **não** casa `axb`.
- `symbol` com fronteira: `Parse` não casa dentro de `ParseTOML` nem de `parseMode`.
- `kind: "def"` num arquivo `.go` casa `func Parse(` e não casa a chamada `Parse(`.
- Extensão desconhecida devolve resultado com `kind` desconhecido, **nunca** erro.
- Todo resultado carrega a declaração de limite — varredura da saída.
- Truncamento declarado, como em `grep` (RN-5).
- Ordenação estável entre execuções e entre máquinas.

## Impacto

- Oitava ferramenta; `glob` e `grep` não mudam de comportamento.
- Tabela de padrões de declaração por extensão, como dado do pacote.
- Descrição de `grep` passa a apontar `symbol` quando a busca é por identificador — a camada de alta precisão, lida no momento da decisão.
- Uma chave nova.
