# A conta que faltava no teto de rodadas

**Data:** 2026-08-30
**Specs afetadas:** `202608301200-recoverable-cycle` ganha o contrato que mede a
premissa do passo seguinte. Nenhuma mudança de produto.

> **Estado.** `verifiedCycleRounds = 32` para cenários que rodam o ciclo de
> verdade, e as duas medições feitas sob ele.

## Um cenário que roda ciclo custa rodadas que o trabalho não vê

Cada ciclo cobra uma rodada que não é trabalho: o modelo tem de **parar de
chamar ferramenta** para o ciclo rodar, e é essa rodada silenciosa que entrega o
lembrete. Um cenário cujos critérios não são todos conhecíveis pela tarefa leva
três ou quatro ciclos, e três ou quatro rodadas vão para a máquina antes de
qualquer uma ir para o trabalho.

`exploreThenActRounds = 20` foi escrito quando **nenhum cenário rodava ciclo**.
A conta nunca esteve nele.

O primeiro cenário desta forma mediu **70%**, e cinco das seis falhas eram
execuções ainda trabalhando quando o arcabouço as cortou.

## Por que isto não é ajustar o instrumento ao resultado

Ontem esta mesma suíte recusou subir o teto do `states-unmet-on-stall`, e a
razão continua de pé: lá seria subir até um contrato passar.

Aqui é diferente, e a diferença é verificável: **surgiu uma forma de cenário que
não existia quando a constante foi decidida.** Ela ganha nome próprio —
`verifiedCycleRounds` — com a aritmética no comentário, para as duas não se
confundirem. O `exploreThenActRounds` fica onde está, valendo para o que ele
sempre valeu.

## O que a mudança de constante obrigou

Remedir `fixes-what-the-output-named`, que já tinha 100% sob o teto antigo. Mudar
a constante muda o cenário, e um número que descreve um cenário que mudou é
exatamente o defeito que a tabela de estado existe para impedir.

Voltou **100% de 20**. O número sobreviveu à mudança, e é isso que autoriza
escrevê-lo.

## E o resultado retirou dois passos do plano

`finishes-work-that-takes-more-than-one-cycle` mediu **95% de 20**. Dezenove
execuções atravessaram vários ciclos, descobriram pela saída dois critérios que
a tarefa não menciona, e terminaram com os cinco verdes.

O plano do laço tinha mais dois passos depois deste:

| | passo | estado |
|---|---|---|
| 3 | progresso por aproximação | **sai** |
| 4 | subir o `stallLimit` | **sai** |

Os dois existiam para corrigir a mesma coisa: o laço desistir de trabalho que
anda sem fechar critério. **A etapa 1 desta família já tinha resolvido isso** —
com `Moved` no lugar do booleano, qualquer avanço zera o contador, e um ciclo só
conta como parado se fechar zero critério.

A intuição era verdadeira quando foi escrita, e deixou de ser por causa de uma
mudança feita por outro motivo. Sem medir, os dois passos teriam sido
construídos, teriam funcionado, e ninguém saberia que não precisavam existir.

É a primeira vez nesta sequência que uma medição decidiu **não** construir, e é
o que a RN-8 do piso manda fazer com contrapeso sem peso.
