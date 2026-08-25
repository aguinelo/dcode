# O modo é um nome para um par, e é derivado dele

Os dois eixos — sandbox e política de aprovação — sempre foram ortogonais, e
continuam. O que faltava era o vocabulário por cima deles: ninguém escolhe um
par, escolhe **quanta autonomia** a ferramenta tem. `plan`, `assist` e `auto`
são os nomes dessa escolha, e a §2.1 do `.p` fixa a tabela.

A parte que virou regra (RN-8) é **de onde o nome vem**. A primeira
implementação guardava o nome ao lado do par: a sessão nascia rotulada `assist`
fosse qual fosse o modo com que o motor tinha sido construído. Duas
consequências, e a segunda é a que morde.

A barra passava a exibir o crachá contido sobre uma sessão `full-access` — um
crachá que diz `assist` sobre uma sessão sem fronteira é pior que crachá
nenhum, porque convida a confiar. E `/mode assist`, o comando com que se
instalaria a fronteira de volta, não fazia nada: a sessão comparava o nome
pedido com o nome guardado, achava que já estava lá, e devolvia sem erro e sem
tocar no motor. O caminho de volta ao limite era o que estava quebrado.

O nome passa a ser derivado: a sessão pergunta ao motor em que par ele está e
nomeia o resultado. Um par que não é nenhum dos três — `read-only` que ainda
pergunta é configuração legítima — não recebe nome, em vez de receber o do
vizinho mais próximo. O vazio é resposta, e o cliente já sabe não desenhá-lo.

Junto veio a segunda metade da ortogonalidade: o par **se move junto**. `plan`
é read-only *e* never; escrever um e depois o outro deixa em vigor, entre as
duas escritas, um par que ninguém escolheu. `Engine.SetMode` escreve os dois sob
o mesmo mutex, e — o que a primeira versão errou — **todo** leitor lê os dois
sob ele. A avaliação de uma chamada tinha sido protegida; a montagem de um
filho delegado, não, e é justamente ela que roda enquanto o turno vive, que é
quando a troca chega. Sob `-race` isso é uma corrida, e nenhum teste a via
porque nenhum teste delegava enquanto trocava.

O que fica escrito como invariante é o que os testes agora perguntam ao
comportamento e não ao campo: que o veredito muda depois da troca, que a
delegação a enxerga, e que trocar concorrentemente deixa o par do motor de
acordo com o nome anunciado.
