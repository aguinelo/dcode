# orchestrator-splits-by-theme

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **≥ 85%**

A orquestradora de um projeto recebe trabalho com três temas independentes e o
divide: **um** `delegate` com três itens, um tema por filho, `owns` disjuntos —
em vez de dizer que não pode escrever e pedir à pessoa que troque o modo.

## O relato

Primeiro teste ao vivo da orquestradora (MiniMax-M3, repositório de brinquedo,
"três pacotes independentes, divididos entre agentes"). Ela leu o repositório,
disse que o terminal era somente leitura e que portanto nem ela nem os filhos
podiam escrever, e pediu à pessoa que trocasse o modo do sandbox — o daemon
recusou, corretamente. `delegate` nunca foi chamado.

A causa estava no prompt: nada dizia o que é uma orquestradora. O único texto
sobre a situação era o bloco de somente leitura de uma sessão comum, que manda
dizer o que falta e deixar a pessoa decidir se troca o modo. Ela fez isso.

Os cenários roteirizados não pegavam, porque o modelo roteirizado já sabe chamar
`delegate`. Este contrato é a metade que só um modelo de verdade responde.

## O que se mede

- **O lote:** uma chamada a `delegate` com pelo menos três itens, todo item com
  `owns`, e nenhum `owns` cruzando o de outro item pela leitura do próprio
  produto (`orchestrate.OwnsOverlap`).
- **A recusa que ninguém deu:** a resposta não pede que o modo seja trocado nem
  diz que o trabalho não pode ser feito por ser somente leitura.

## O material

O workspace compartilhado, um módulo Go. O mundo é o da orquestradora: modo
`read-only` e papel `orchestrator` (`world.json`), e as ferramentas do registro
dela — inclusive `write`, `edit` e `bash`, que o modo nega, como no produto.
O projeto por trás de `delegate` é um substituto que aceita todo lote e não roda
filho nenhum: o que se julga é o despacho, que já aconteceu quando um filho
começaria.

## O que não se mede

Se os filhos fazem o trabalho, a esteira, ou a qualidade de cada `task`. E a
divisão de uma tarefa pequena, que é de `orchestrator-does-not-split-the-small`.

## Por que 85%

É o limiar que o `.r` da orquestração propôs para este cenário. Um lote com dois
temas num filho só é discutível; pedir a troca de modo não é.
