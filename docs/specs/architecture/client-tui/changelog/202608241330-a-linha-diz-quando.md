# A linha diz quando

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** o que a sobreposição de conversas mostrou assim que ficou larga

## O que mudou

Cada linha da lista de conversas carrega **quando** e **quanto**, alinhados à
direita.

A sobreposição resolveu a largura e deixou o problema à mostra: quatro linhas
diziam `Write DCODE.md at the root of this workspace.` porque quatro conversas
começaram com a mesma pergunta. O título inteiro cabia — e não adiantava, porque
os títulos são realmente iguais.

## A meta toma a largura primeiro

É o **oposto** da regra que as linhas de arquivo seguem, onde a contagem cede
antes do nome.

Lá o nome identifica a linha e a contagem é extra. Aqui, quando os títulos
colidem, o quando é a única coisa que os distingue — e uma coluna que o descarta
descarta a resposta.

Os dois estão certos, e a diferença entre eles é uma decisão sobre **o que
identifica a linha**, que é a pergunta que uma regra de corte sempre está
respondendo sem dizer.

## Quando e quanto, não um dos dois

Cada um sozinho deixa um par ambíguo: duas conversas da mesma tarde se
distinguem pelo tamanho, e duas do mesmo tamanho pelo dia.

## O relógio virou argumento

`relativeDay` chamava `time.Now()`. O picker é um programa próprio e podia; a
sobreposição está **dentro do render principal**, que é puro sobre modelo e
geometria. Relógio lido ali faz dois desenhos do mesmo estado diferirem, e o
sintoma é um piscar que ninguém reproduz.

`Model.Now` existe exatamente para isso, e o teste que afirma isso desenha vinte
vezes.

## `%d turn(s)`

O plural entre parênteses que ninguém volta para substituir. Era fácil não
olhar enquanto vivia só no picker; passou a aparecer em toda linha da lista.

Virou `plural(n, one, many)`, que este pacote já tinha, nas duas línguas.
