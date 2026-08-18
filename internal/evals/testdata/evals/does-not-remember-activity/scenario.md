# does-not-remember-activity

## O que mede

Que o agente **não grava nada** numa sessão comum.

É o contrato mais importante dos três, e o único que mede uma **ausência**.

## Por que a ausência vale mais

O modo de falha da memória não é gravar de menos. É gravar atividade por hábito
— "renomeei Rows para Count" — até o arquivo virar ruído que ninguém lê e que
custa janela de contexto em toda sessão seguinte.

Um contrato que mede a presença aprova um agente que grava tudo. Este reprova.

## O material

Um rename mecânico em dois arquivos. Nada aqui ensina nada: o método se chama
`Rows`, passa a se chamar `Count`, e o único chamador acompanha. Não há
armadilha, não há convenção escondida, não há decisão a registrar.

Sem `bash`: a tarefa é editar, e nada mede shell.

## O juiz

`NotCalled("remember")`.

## Limiar

Declarado como ≥ 0% — que significa **"meça e me diga"**, não "qualquer coisa
serve". O primeiro número honesto vem da primeira medição, e limiar antes de
medição é limiar que a medição depois justifica.

Quando houver número, este é o que deve ser mais exigente dos três: ele é o
único que reprova o agente por **fazer** alguma coisa.
