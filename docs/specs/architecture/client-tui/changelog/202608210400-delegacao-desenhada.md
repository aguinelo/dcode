# Delegação é desenhada como um card com os filhos dentro

**Data:** 2026-08-21
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** `refs/design/HANDOFF.md` (v5, §4)

## O que mudou

Chamadas `explore` adjacentes passam a ser um bloco só: um cabeçalho
`⏺ explore · 4 filhos`, e uma sub-linha por filho com o glifo, o nome, `possui
<caminho>` e o resultado. Quem não respondeu é nomeado na própria linha, com o
motivo, e contado no cabeçalho.

## Por que isto pesa mais que um card

O `delegated-writing` promete que **filho que não respondeu é nomeado, com o
motivo, nunca resumido junto dos que responderam**. Essa garantia valia no laço e
sumia na tela: o cliente desenhava um `explore` que falhou exatamente como
qualquer outra chamada que falhou — uma linha entre irmãs, onde ela se lê como a
sexta linha e não como *a* linha.

Garantia que vale numa camada e desaparece na camada que a pessoa olha é a forma
que este repositório não para de encontrar, com outro nome.

## Não precisou de protocolo

A primeira leitura foi que precisaria. O `delegate.go:221` diz, deliberadamente,
que *"os passos do filho não são eventos da sessão do pai"*, e o protocolo não
tem campo nenhum sobre filhos.

Mas **cada filho é uma chamada `explore` própria**, então os quatro filhos já
chegam como quatro entradas. O que faltava era o `owns`, que vem no **input** da
chamada e estava sendo descartado por `targetOf`. É agrupamento no cliente, não
superfície nova.

Adjacência é o sinal, porque os eventos chegam em ordem de emissão e um lote
delegado é emitido junto. Duas delegações separadas no mesmo turno se leriam como
uma — e é o jeito certo de errar: ainda foram uma decisão de repartir trabalho, e
os filhos continuam com os nomes deles.

Um filho só não vira card. Moldura e contagem em volta do que a linha comum já
diz é ruído.

## Um defeito que a tela mostrou

Os nomes dos filhos — `alpha`, `bravo` — **entraram na coluna ARQUIVOS** como se
fossem arquivos. Nome sem pontuação parece nome de arquivo, e o `looksLikePath`
não tinha como saber.

A leitura que não se engana é por **ferramenta**, não pelo formato da string:
`explore`, `bash`, `process`, `plan` e `remember` não nomeiam caminho.

## O que continua faltando do §4

A barra de progresso por filho. É o mesmo evento `tool.progress` que falta para o
card comum, e está no `docs/ROADMAP.md`.
