# Os portões que o projeto declara

**Data:** 2026-08-27
**Specs afetadas:** `202608080016-behavior-definition` — nove invariantes
novas na §8 e uma chave nova no `.config`. Implementa a etapa 3 da
`202608262200-working-defaults.p §8`.

## O defeito

Um projeto auditado declarava quatro portões. **Dois estavam vermelhos desde o
primeiro dia** — `lint` quebrado por ferramenta depreciada, cobertura em 0%
contra um piso de 80% que o próprio arquivo do projeto chamava de "mínimo pra
merge". O terceiro passava verde medindo `1 + 1 === 2`.

Ninguém rodou nenhum. Para saber que existiam era preciso abrir o
`package.json` — e um fato que exige consulta é um fato usado quando alguém
lembra dele.

## O que passou a existir

`internal/workspace.Probe` lê `package.json` e `Makefile`, devolve `[]Gate`, e
o bloco do workspace passa a nomeá-los:

```
This project declares its own checks:

  lint           next lint
  test           vitest run
  test:coverage  vitest run --coverage
  typecheck      tsc --noEmit

These are what the project measures itself by. Nothing here says they pass,
and nothing has run them.
```

## A última frase é a peça

Sem ela, uma lista de portões no prefixo lê como uma lista de **garantias** — e
esta seção teria produzido exatamente o defeito que a pediu.

Ela é constante não configurável e tem invariante própria. Nomear não é medir:
medir é `202608261730-done-qualifier`, e a regra do vermelho inicial mora lá.

## Por que um pacote novo

`internal/vcs` lê git e diz isso na primeira linha. Portão declarado é
`package.json` e `Makefile`, que não têm nada a ver com controle de versão;
enfiar ali criaria um pacote chamado `vcs` que lê `package.json`.

O tipo `Workspace` mora no `behavior/` pela mesma razão que `Repo`: `Build` é
pura e nada dentro dela toca disco. O `behavior/` define **o que o prefixo
carrega**; o `workspace/` descobre **o que há para carregar**. `Gate` é
espelhado em vez de importado — um alias seria um import.

## Ordenação, porque o prefixo é cacheado

Os scripts saem em ordem alfabética e os alvos de `Makefile` em ordem de
aparição.

Alfabética porque Go randomiza iteração de mapa, e um prefixo que se
reembaralha entre execuções invalida o cache do provider em toda sessão, de
graça. Ordem de aparição no `Makefile` porque ele é escrito para ser lido de
cima para baixo e o primeiro alvo é o default.

Há invariante para as duas coisas, e uma delas roda dez vezes a mesma sonda.

## O que um `Makefile` carrega e não é portão

`.PHONY`, `.DEFAULT_GOAL`, `VERSION := 1.0`, `FLAGS = -race`, `%.o: %.c`,
`$(BIN): main.go` — diretivas para o `make` e regras de padrão, não coisas que
uma pessoa roda. Nenhuma vira `Gate`.

## O bloco mudou de nome

`This repository` virou `This workspace`.

O bloco já carregava, desde a mudança de ontem, a linha que diz que **não há**
repositório — e um cabeçalho dizendo "este repositório" acima de uma linha
dizendo que não há um lê como contradição. Agora ele carrega duas classes de
fato sobre o workspace, e o nome cobre as duas.

## A lição do F-2, terceira aplicação

Projeto que não declara portão **não gera seção**, e nada no prefixo afirma que
ele não declara nenhum. Sonda cancelada devolve nada. Invariantes para as duas.

E a diferença com o repositório ausente está escrita no código: não ter
repositório **muda o que terminar significa**, e por isso vale uma linha; não
declarar portão é comum e sem consequência, e por isso não vale. **Nem toda
ausência é digna de nota — o que não pode é ausência não conferida virar
afirmação.**

## A chave

`DCODE_WORKSPACE_GATES`, ligada por default: a sonda lê dois arquivos, roda
nada, e custa uma leitura na abertura. Existe para o repositório com `Makefile`
de setenta alvos, onde o teto de vinte ainda deixa lista que ninguém lê.

Ela entrou em `KnownKeys` **e** na tabela do `.config` no mesmo commit, porque
foi este commit que passou a lê-la. As duas guardas cobraram isso — a de wiring
pediu a linha que diz quem lê, e a de spec pediu a linha de tabela — e as duas
estavam certas.
