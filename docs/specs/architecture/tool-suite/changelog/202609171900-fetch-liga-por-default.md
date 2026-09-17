# `fetch` liga por default

**Data:** 2026-09-17
**Specs afetadas:** `202608072337-tool-suite` (`.r`, RN-13; `.config`, `DCODE_FETCH_ENABLED`)
**Fonte:** pedido do usuário — "tirar limite de rede, acesso a rede livre, pode
pesquisa na web por default, restrição somente como já existe para escrita".

## O que a RN-13 dizia, e por que parou de valer

`fetch` nasceu desligada por duas razões, na mesma frase: "rede sem modelo de
permissão é buraco", e é a única capacidade cuja ausência ninguém precisa
contornar — as outras oito ferramentas são necessárias pro trabalho, `fetch`
não é. A segunda continua verdadeira. A primeira não é mais: o parágrafo
logo acima da regra, na mesma spec, já descreve o modelo de permissão —
`grants.toml`, o eixo de política pra rede, o avaliador que recusa o que não
foi concedido — e esse modelo já trata `sandbox.allow_network` como
concedido por default no `workspace-write`, sem pedir aprovação por
travessia. Rede, em geral, já não pedia. `fetch` era a única ferramenta do
conjunto pedindo mais que o resto sem que a razão original ainda estivesse
de pé.

## O que muda

`tools.fetch_enabled` (`DCODE_FETCH_ENABLED`) vai de `false` pra `true` como
default, em `internal/app/app.go` (a camada default de `Resolve` e o
`r.Bool` de `fromResolved`). Nada na ferramenta em si muda: continua correndo
fora do sandbox do SO (RN-13, `roda neste processo, não atrás do sandbox`),
continua recusando corpo binário, continua carregando o endereço de origem
na resposta. A garantia de segurança sempre foi o veredito da política, não
o sistema operacional — isso não muda com o default, porque o default nunca
foi a garantia.

`--fetch-enabled false` / `DCODE_FETCH_ENABLED=false` continua desligando a
ferramenta pontualmente, e `sandbox.allow_network=false` continua bloqueando
a rede inteira, `fetch` incluído — os dois eixos de recusa que já existiam
seguem intactos; só o ponto de partida mudou.

## Por que não foi preciso mexer no sandbox ou na política

A pergunta original citava "restrição somente como já existe para escrita"
como o alvo. Investigando antes de mexer: essa restrição **já existe**, e é
exatamente a de escrita — `internal/policy/policy.go`, `evaluateMode`,
concede rede sem perguntar quando `net.Granted()` é verdadeiro em qualquer
modo além de `read-only`, o mesmo comentário no código nomeando o motivo:
"perguntar sobre a declaração significaria perguntar sobre todo build, todo
teste, todo commit." Não havia sandbox ou política pra afrouxar — só um
default de ferramenta que não acompanhou a mudança de premissa.
