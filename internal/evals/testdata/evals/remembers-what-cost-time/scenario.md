# remembers-what-cost-time

## O que mede

Que o agente **grava** uma `gotcha` depois de descobrir, com custo, uma coisa
que este repositório ensina e ninguém escreveu.

## O material

A tarefa pede um rodapé no relatório. Para escrever um, é preciso uma string
nova — e as strings do relatório moram em `generated.go`, cujo cabeçalho diz
**DO NOT EDIT**.

Não há gerador neste checkout. O `README.md` explica que o arquivo é mantido à
mão e que `schema.yml` acompanha, e que nada obriga esse par.

Ou seja: **a descoberta está no caminho crítico**. Não dá para terminar a tarefa
sem resolver "posso editar um arquivo que diz para não editar, e por quê" — que
é uma coisa que custa rodadas, não é óbvia, e é exatamente do tipo que vale
guardar.

## O que as duas primeiras versões erraram

**A primeira** pedia para consertar uma build cujo único reparo era um
`make generate` que a fixture não tinha. Cinco de seis execuções gastaram as
doze rodadas procurando.

**A segunda** deixou a armadilha **ao lado** da tarefa em vez de dentro dela, e o
modelo fez a leitura certa:

> *"`report.go` usa `Label()` que não existe — isso é um erro de compilação
> pré-existente, não relacionado ao meu trabalho."*

Concluir que um erro pré-existente não é seu trabalho é defensável. Para gravar
uma memória ser natural, a descoberta tem de ser algo que o agente **precisou
resolver para terminar**, não algo que pode anotar e contornar.

**A terceira** — e este é o erro que mais se repetiu — usava `package example`
enquanto o workspace compartilhado é `package stats`. O modelo gastava rodadas
numa incoerência que a fixture criou.

## O juiz

`CalledWith("remember", "gotcha")`.

Mede a chamada e o tipo, não o texto. Julgar a redação seria medir a formulação
em vez do comportamento, que é o defeito que esta suíte já corrigiu em quatro
juízes.

## Limiar

Nenhum, ainda. O primeiro número honesto vem da primeira medição. Limiar antes
de medição é limiar que a medição depois justifica.
