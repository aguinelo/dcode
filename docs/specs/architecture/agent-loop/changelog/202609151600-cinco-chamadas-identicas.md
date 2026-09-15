# Cinco chamadas idênticas

**Data:** 2026-09-15
**Specs afetadas:** `202608072335-agent-loop` (`.p`, `.config`), `202608252000-loop-command` (`.p`)
**Fonte:** pedido do usuário — na mesma conversa que revisou o teto de
rodadas da família `generic`, pediu explicitamente que "5 chamadas idênticas
encerra o processo com alerta".

## O que mudou

`MaxIdenticalCalls`: default 3 → **5**. `DCODE_MAX_IDENTICAL_CALLS`, `--max-identical`
e `loop.DefaultLimits()` seguem a mesma mudança — um número, três lugares
que já o citavam, nenhum deles com deriva entre si.

"Com alerta" já estava coberto por outra correção da mesma leva
(`docs/specs/architecture/client-tui/changelog/202609151400-o-turno-que-parava-mudo.md`):
`StopRepeatLoop` — o motivo que este detector emite — agora produz uma nota
visível no cliente em vez de terminar em silêncio. Nenhuma tela nova aqui,
só o número que decide quando o motivo dispara.

## Por que 5 e não outro número

Cinco continua "bem antes de qualquer teto de iteração razoável" — a
premissa que já justificava três, intacta com o número maior. O motivo de
subir não foi achar três frágil; foi o mesmo turno curto de conversa que
levantou os outros dois pontos (o teto de `generic`, o silêncio do
cliente), e cinco é a margem que o usuário pediu diretamente, sem uma
medição nova por trás — vale registrar que não é número medido, é decisão.

## O que não mudou

O mecanismo continua igual: `IsRepeat` compara as últimas N chamadas por
igualdade exata — mesma ferramenta, mesmo input canonicalizado — e N é a
única coisa que mudou. Segue sendo, como já era, o mecanismo real contra
loop patológico, independente do teto de iterações de qualquer família.
