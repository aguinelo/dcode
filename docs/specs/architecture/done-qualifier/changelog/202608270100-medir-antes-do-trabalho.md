# Medir antes do trabalho

**Data:** 2026-08-27
**Specs afetadas:** `202608261730-done-qualifier` — a §1 passa a "parcialmente
entregue", as invariantes se dividem em verificáveis (§9) e previstas (§10), e
duas assinaturas da §5 mudam. Implementa a etapa 1 da §12.

## O que passou a existir

`internal/loop/qualifier`: `Proposal`, `Proposed`, `Expectation`, `Class`,
`Measured`, `Conditions` e `Measure`.

Puro no que importa: nada escreve, nada deriva critério, nada fala com modelo
nem com operador. O runner é **injetado**, e é o mesmo `loop.CriterionRunner`
que o laço usa — o que roda aqui passa pelo sandbox exatamente como vai passar
depois.

100% de cobertura no pacote, e não por perseguir número: a superfície é uma
função e um switch de três casos.

## A regra, nos dois sentidos

Critério que **falha** antes do trabalho é aceitação: ele pode testemunhar que
o trabalho aconteceu. Critério que **passa** é guarda de regressão: o trabalho
dele é continuar verde.

As duas classes são legítimas por motivos **opostos**, e é por isso que a
classificação não filtra — ela classifica. Há teste para as duas metades na
mesma função, porque separá-las convidaria alguém a implementar uma e achar que
implementou a regra.

## Os dois detalhes que decidem se isso funciona

**`Exit == ExitCode`, nunca `Exit == 0`.** Um critério declarado com `exit: 1`
está cumprido saindo 1. Comparar com zero classificaria como aceitação um
critério que já está verde — precisamente o erro que o pacote existe para não
cometer.

**126 e 127 são `ClassBroken`, e falha ao iniciar entra junto.** Um comando
inexistente falha, e falhando se disfarça de aceitação. Ele mede a ausência da
ferramenta, não a ausência do trabalho, e ficaria vermelho para sempre.

Um critério que estourou o prazo também é quebrado, não vermelho — pela mesma
razão: não se sabe nada sobre o trabalho a partir dele.

## Duas mudanças de contrato em relação ao `.p`

**`Measure` devolve `Conditions`.** Era `([]Measured, error)` com `Conditions`
solto. Devolver junto é o que garante que quem mediu vê a condição; um tipo que
o chamador precisa lembrar de calcular é um tipo que ele calcula quando lembra.

**`Conditions.Empty` virou `ErrEmptyProposal`.** Uma condição só é observável
se a chamada devolve o conjunto, e proposta vazia não devolve conjunto nenhum:
ela para ali. Como campo, `Empty` seria verdadeiro num valor que o chamador
nunca receberia — um campo que só existe na prosa.

O conteúdo da regra não mudou: proposta sem critério é **erro**, nunca `DoneSet`
vazia. Vazia significa "não há o que verificar", que o ciclo relata como pronto.

## O que a medição nomeia e não decide

Conjunto sem nenhum critério vermelho é **nomeado** (`NoAcceptance`) e nunca
recusado. Vai relatar pronto sem que nada precise mudar, e quase sempre isso é
defeito — mas uma refatoração legítima é exatamente isso, nada de novo a provar
e tudo a preservar.

O harness não sabe distinguir as duas, então não decide. Decidir seria ele
escolhendo o que conta como medição.

## A discordância

`Proposed.Expects` não decide nada — a classe vem da medição. O que ele produz
é a **discordância**: o proponente disse que falharia e passou.

Sem ela, "critério 2 passou" é fato neutro. Com ela, é fato contra o que foi
declarado, que é a assinatura exata de um critério que não mede o que deveria.

`ClassBroken` **não** é discordância: é alarme próprio, e vale independentemente
do que o proponente esperava.

## Um campo exportado que ninguém lê ainda

`Proposed.Why` — a linha que o proponente escreve para o humano que assina.
Nenhum código fora de teste a lê, porque o leitor é a superfície de assinatura,
que é a etapa 2 e não existe.

A guarda de nomes exportados cobrou, e a isenção está escrita: apagar o campo
significaria o operador revisando uma lista de comandos sem uma palavra sobre
para que serve cada um.

## O que isto ainda não faz

Não deriva critério, não pergunta nada a ninguém, não congela `DoneSet` e não é
alcançável pelo produto. É a metade determinística, entregue sozinha porque não
depende de nenhuma decisão que as outras etapas ainda tomam.

A etapa 2 — medir as origens que já existem e o ida-e-volta da assinatura — é a
que entrega valor visível, e é a que produz o número da §11: **com que
frequência um critério proposto já está verde?**
