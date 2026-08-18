# remembers-what-cost-time

## O que mede

Que o agente **grava** uma `gotcha` depois de descobrir, com custo, uma coisa
que este repositório ensina e ninguém escreveu.

## O material

`report.go` chama `Label()`, que não existe. `generated.go` diz, no cabeçalho,
que veio da versão 1 do `schema.yml`; o `schema.yml` no disco é a versão 2 e
declara `Label`. O `README.md` explica que os identificadores gerados vêm do
schema e que o erro aponta para quem chama, não para o arquivo velho.

Ou seja: a causa é descobrível **lendo**, e não é óbvia. Custa rodadas juntar
três arquivos para chegar nela — que é exatamente o que uma gotcha é.

## Sem shell, de propósito

O harness recusa comando de shell, então um cenário de "a build quebrou" que
dependesse de rodar a build mediria a recusa. A causa aqui se descobre lendo, e
por isso `bash` não é oferecido — a lição que este projeto já pagou uma vez,
quando onze fixtures carregavam um shell que nada media e cada corrida gastava a
primeira chamada nele.

## O juiz

`CalledWith("remember", "gotcha")`.

Mede a chamada e o tipo, não o texto. Julgar a redação seria medir a formulação
em vez do comportamento, que é o defeito que esta suíte já corrigiu em quatro
juízes.

## Limiar

Nenhum, ainda. O primeiro número honesto vem da primeira medição. Limiar antes
de medição é limiar que a medição depois justifica.
