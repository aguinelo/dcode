# O texto tem hierarquia

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** recomendação de redesenho — a paleta como slots semânticos

## O que mudou

`StyleDim` significava cinco coisas em quarenta e sete lugares. Agora são seis
papéis, e o mapeamento é **uma decisão numa tabela só**.

E a primeira coisa que essa decisão mudou: **a prosa deixou de ser apagada.**

## A resposta era a coisa mais apagada da tela

`parseInline` desenhava a frase em `StyleDim` e o termo técnico dentro dela em
brilho normal. O raciocínio está escrito e é bom: o olho pousa no nome do
arquivo, não na frase em volta.

Só que a prosa do modelo é **a maior parte do que está na tela**. Apagar a frase
apagou a tela, e a única coisa pela qual o leitor veio foi a única que ficou
desbotada.

O contraste dentro da frase continua sendo comprado — com o **termo**, e não com
o parágrafo inteiro. Uma palavra em vez de um parágrafo.

## Seis papéis, três pesos

| Papel | O que é | Peso |
|---|---|---|
| `StyleProse` | a frase que se lê | normal |
| `StyleCode` | termo técnico dentro dela | âmbar |
| `StyleHeading` | rótulo de seção | negrito |
| `StyleMeta` | fato que qualifica outro | SGR 2 |
| `StyleHint` | tecla que alguém pode apertar | SGR 2 |
| `StyleChrome` | régua, calha, moldura | SGR 2 |

## A escala de cinco degraus do desenho não sobrevive

A recomendação falava de uma escala neutra com `text.primary`, `text.muted`,
`text.emphasis` e mais. Confrontada com o terminal, ela encolhe: **existem três
pesos que sobrevivem a um fundo desconhecido** — negrito, normal e SGR 2.

O quarto que se costuma alcançar é um cinza cravado, e um cinza escolhido para
tema escuro é ilegível em tema claro. Isso está afirmado como invariante, não
como intenção.

Três dos seis papéis compartilham o SGR 2 hoje, e está certo que compartilhem:
o ganho não é um quarto tom, é que quando houver um, ele se escolhe num lugar só
em vez de em quarenta e sete.
