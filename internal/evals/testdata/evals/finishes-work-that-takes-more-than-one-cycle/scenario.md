# finishes-work-that-takes-more-than-one-cycle

**Contrato:** `202608301200-recoverable-cycle.p.spec.md` · limiar **80%**

Mede a **premissa** do passo seguinte antes de construí-lo.

## A pergunta

O plano do laço tinha como terceiro passo "progresso por aproximação": a ideia
de que o `stallLimit = 2` é apertado e faz o laço desistir de trabalho que está
andando sem fechar critério.

Depois que a `recoverable-cycle` trocou o booleano por `Moved`, essa premissa
ficou em dúvida: **qualquer avanço zera o contador**, e um ciclo só conta como
parado se fechar zero critério. Então o teto talvez nunca morda.

Construir contrapeso sem o peso é o que a RN-8 do piso proíbe, e a resposta é
uma medição em vez de uma opinião.

## O material

Cinco critérios em três arquivos, e dois deles **não são adivinháveis** pela
tarefa: ela pede só a correção do `Slugify`. Que exista um teste cobrindo
espaços e um `doc.go` com o comentário de pacote só aparece quando o primeiro
ciclo falha e a saída do critério diz o que falta.

É o cenário mais próximo do trabalho real que esta suíte tem: a régua sabe mais
que o pedido, e descobrir isso é o ciclo funcionando.

## O que se mede

Que os cinco fiquem verdes. Como pelo menos dois só são conhecíveis depois do
primeiro ciclo, passar exige **mais de um ciclo** — e exige que o laço não
desista no meio.

## Como ler o resultado

**Alto**: o teto não morde, e os passos 3 e 4 do plano saem — não há defeito
para eles corrigirem, que é exatamente o que a RN-8 manda fazer com um
contrapeso sem peso.

**Baixo, com falhas por ciclo parado**: a premissa está certa, e aí o passo 3 se
justifica com evidência em vez de intuição.

**Baixo, com falhas por teto de rodadas**: não mede nada sobre o laço, e o
cenário precisa de mais rodadas antes de valer qualquer conclusão.
