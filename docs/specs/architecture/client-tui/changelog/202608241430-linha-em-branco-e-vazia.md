# Linha em branco é vazia

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** quadro renderizado a partir da sessão real `1a030dd642cd50b9f68`

## O que mudou

Prosa parava de deixar duas e três linhas em branco entre parágrafos, e a linha
em branco passou a ser vazia.

## O invariante já existia e a guarda já trimava

**Nunca duas linhas em branco seguidas** está declarado, e
`TestTwoBlocksAreSeparatedByExactlyOneBlankLine` já comparava com `TrimSpace`.

O que ela nunca viu foi **prosa**. O fixture era feito só de chamadas de
ferramenta, e a regra é sobre o fluxo inteiro.

Então a correção é do teste antes de ser do código: prosa entrou no fixture, e
com ela o defeito apareceu na primeira execução.

## Duas causas, três linhas em branco

`strings.Split("a\n\n", "\n")` tem **três** partes, e a última é o fim do texto,
não um parágrafo. Um `\n\n` no fim produzia duas linhas em branco.

E `parseInline` corta o texto em runs: `antes:\n\n**título**` é um run comum e um
run em negrito, cada um dividido por conta própria. O bloco saía com **três**
linhas em branco entre duas frases.

Colapsar mora no `wrapStyled`, e não em quem chama: quem chama vê linhas, e este
é o único lugar que sabe quais delas eram quebras de parágrafo.

## E elas eram espaço, não vazio

O fluxo escrevia `"  " + linha` em toda linha de prosa, quebra de parágrafo
inclusive. A linha em branco virava dois espaços.

Toda regra sobre linha em branco neste pacote compara com `""` ou trima. As duas
leituras passavam por cima de dois espaços enquanto prosa não fosse testada — e
esse é o motivo de o defeito ter atravessado meses de guarda verde.
