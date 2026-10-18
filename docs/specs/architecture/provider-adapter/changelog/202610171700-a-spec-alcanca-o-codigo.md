# A spec alcança o código

**Data:** 2026-10-17
**Specs afetadas:** `202608072334-provider-adapter` (`.r`, §1 e §6; `.p`, §1,
§2, §2.1, §3 e §5; `.config`, §2)
**Fonte:** revisão das specs contra o código. Só documentação: nenhum
comportamento muda.

## O que estava desatualizado

- **A `.r` dizia "no MVP existe uma implementação".** Existem quatro famílias
  — `minimax-m3`, `claude`, `gemini` e `generic` — sobre dois transportes. E a
  lista de changelog da `.r` citava dois dos dezessete registros da família.
- **O `.p` §1 mandava avaliar a promoção a `pkg/provider` "quando uma terceira
  família existir".** Existe a quarta e a avaliação não foi feita. Fica
  escrito como pendência, não como decisão.
- **A tabela 2.1 tinha duas famílias.** Ganhou `gemini` e `generic`, e uma
  segunda tabela com janela, imagens, PDF e se a família foi medida.
- **Os eventos de stream estavam incompletos.** Faltavam
  `EventReasoningDelta`, `EventToolCallOpened` e `EventToolCallProgress`, os
  campos `CallID`, `CallName`, `Bytes` e `Truncated`, e o `Usage` no
  `EventError` da recusa. O `Decoder` não tinha `Close()`, e `WireRequest` e
  `WireEvent` não estavam escritos.
- **O `Registry` estava com a assinatura de antes.** `RegisterFamily` devolve
  `error` — é ali que a sobreposição de prefixo é recusada —, e `ResolveFamily`,
  `FamilyFor`, `FamilyNames` e `TransportNames` existem e não constavam.
- **A `.config` prometia a variável convencional de credencial de cada
  família** (`ANTHROPIC_API_KEY` e afins). O código não lê nenhuma: a chave vem
  de `DCODE_API_KEY` ou do armazenamento por família. A frase virou nota de
  "não implementado", com o caminho do código.

## O que fica registrado sem conserto

**Aliases Claude sem sufixo caem no default.** `claude-sonnet-4` e
`claude-opus-4` não casam com nenhuma linha de `claude_models.go` — as linhas
da geração 4 são `-4-0` e `-4-2025` — e recebem 200.000 de janela e 32.000 de
teto. Para `claude-opus-4` é o número publicado; para `claude-sonnet-4` o teto
fica na metade dos 64.000 que o modelo aceita. É defeito de código, e uma
correção documental não o conserta: fica no `.p` §2.1 como lacuna conhecida.

## O que foi conferido e já estava certo

As entradas do erro Anthropic classificado (`202610161630`) e da recusa
(`202610162300`) já estavam no `.p` §4 e §7 e batem com `events.go` e
`anthropic_errors.go`.
