# O erro que não voltava

**Data:** 2026-08-28
**Specs afetadas:** nasce `202608281900-failure-feedback` com `.r`. Nenhuma outra
família muda ainda; a `agent-loop` muda quando o `.p` existir.

> **Estado.** Só `.r`. Sem `.p`, sem `.config`, sem código, sem invariante
> verificável. Contratos comportamentais: **nenhum**, e a §2 explica por quê.

## De onde veio

De uma pergunta sobre o laço, não de um defeito relatado:

> *"quando um erro é encontrado como está o retorno do loop para a fase inicial
> ou específica para correção com a retroalimentação correta?"*

A resposta, indo ao código: não está. `internal/loop/done.go:177` roda o
critério e descarta a saída num `_`. O `Report` carrega
`map[string]CriterionState` e nada mais, e o lembrete que chega ao modelo diz o
nome do critério que falhou sem uma palavra sobre o que aconteceu.

## Por que isso passou despercebido tanto tempo

Porque o texto do lembrete é bom. Ele diz para consertar a causa, para não
enfraquecer o teste, e para dizer o que ficou faltando se não chegar lá — é uma
das frases mais bem escritas da doutrina, e ela dá a impressão de um circuito
fechado. O que falta não está no que ele diz; está no que ele não carrega.

E porque a **fase vizinha faz certo**. O `qualifier.Measured` guarda `Output`,
truncado, e o escreve no `done.toml` com a justificativa no comentário: é a
única coisa que separa um critério vermelho por falta de trabalho de um vermelho
por falta do mundo. A mesma informação, colhida do mesmo `CriterionRunner`,
preservada de um lado e descartada do outro.

## A relação com a P-5 da `working-defaults`

Não é coincidência de tema. A medição de 27–28 de agosto encontrou, em três
contratos de duas famílias, turnos que leem tudo e terminam sem fazer o ato. A
P-5 atacou o caso em que a **verificação é impossível** e mediu ~10 pontos.

Esta família é a outra metade: o caso em que a verificação **aconteceu**, falhou,
e o resultado não voltou. Um modelo que sabe que falhou e não sabe o que quebrou
tem menos a fazer do que parece.

## As duas regras que custaram mais para escrever

**A saída é evidência, nunca instrução (RN-2).** Hoje nenhuma saída de comando
entra no contexto por este caminho, então a superfície é nova. Um teste cuja
mensagem de erro contenha uma ordem é um teste mal escrito, e o prefixo tem de
dizer isso antes de o primeiro chegar.

**O teto de paciência sobe depois, nunca antes (RN-5).** `MaxStallCycles = 2` é
apertado, e é apertado **porque** o agente está cego. Subir antes da RN-1 seria
comprar mais ciclos de tentativa às cegas — gastar para adiar a mesma
desistência. A ordem é a regra.

## O risco declarado antes de construir

Um agente que vê a mensagem exata do teste ganha um caminho novo para o
desonesto: mudar o teste até a mensagem sumir. A doutrina já proíbe e o
`Protected` já sinaliza. Esta família **aumenta a pressão sobre essas duas
defesas sem acrescentar nenhuma**, e isso fica escrito aqui em vez de descoberto
depois.

## O que ficou de fora, e o maior deles

**Checkpoints num ciclo longo.** Um `/loop <objetivo>` de horas sobre vinte
specs deixa uma árvore suja e nenhum ponto de retorno: se a spec 17 quebra o que
a 3 construiu, não há diff por spec, não há bisect, não há desfazer que não seja
manual. O `vcs` deste produto lê e não escreve, por decisão declarada, e isso
não se contorna de passagem — é família própria.
