# A medição que não sabe o que viu

**2026-09-01** — uma medição passa a registrar qual prompt ela viu; a tabela de
estado conta quantas não sabem.

## O achado

Ao decidir o que remedir depois da semana de mudanças, a pergunta óbvia era
"quais contratos foram afetados". A resposta foi: **todos**.

O bloco de skills passou a ser renderizado em toda sessão, inclusive nas que não
têm skill nenhuma. Os prompts de eval o carregam:

```
prompt bytes=4420  has '## Skills'=true  has 'None are installed'=true
```

Ou seja: as dezenove medições registradas viraram, em silêncio, descrições de um
produto que não existe mais. E **nada conseguia perceber**.

## O mesmo defeito, um nível acima

Este repositório tem uma linha na tabela de estado — "contratos de fato já
medidos" — que existe porque um número copiado de uma verdade que se moveu já
enganou aqui. A guarda que a conta impede o número de envelhecer.

O que ela não vê é a medição envelhecendo. Não um número velho **sobre** as
medições: medições velhas. O `Measurement` registrava modelo e data, e o
comentário do campo `Model` explica por que — *"um limiar medido contra um modelo
não diz nada sobre outro"*. A frase estava pela metade. Um limiar pertence a um
modelo **e a um prompt**, porque o prompt é através do que o modelo é medido.

## O que passa a existir

`Measurement.Prompt`: doze caracteres da soma do prefixo contra o qual a
medição rodou. O relatório do eval os imprime junto do número, então registrar
uma medição é copiar uma linha e não um segundo passo manual que ninguém faz
duas vezes.

Vazio significa **não registrado**, e é a resposta honesta para tudo o que foi
medido antes de o campo existir. Não é sinônimo de "atual" — e a linha nova da
tabela conta exatamente essas: dezenove de dezenove.

## Reportado, não imposto

Fazer mudança de prompt reprovar o build significaria todo PR de prompt
carregando cinquenta e três remedições. Regra que ninguém consegue pagar é regra
que alguém desliga, e guarda desligada não protege nada.

O que isto compra é a distância ficar **visível** — exatamente o trabalho que a
linha "de fato já medidos" faz para a distância entre declarado e verificado.

## Duas coisas que a guarda de exportação pegou

Escrevi uma função `Stale` para comparar impressões digitais e **nada a
chamava**. Código escrito para depois, que é o que aquela guarda existe para
impedir; apagada. E a `Unverifiable` só era lida por teste do próprio pacote —
passou a ser interna.

## Invariantes

- `TestEveryMeasurementSaysWhichPromptItSawOrAdmitsItCannot`
- `TestTheFingerprintMovesWithThePromptAndNotOtherwise`
- `TestAnUnrecordedFingerprintIsNotReadAsCurrent`
- `TestTheStateTableIsCountedAndNotCarried` passa a conferir a linha nova, nas
  duas edições.
