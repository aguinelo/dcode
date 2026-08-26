# A definição de pronto passa a ter uma fase que a levanta

**Data:** 2026-08-26
**Specs afetadas:** nova família `202608261730-done-qualifier` (só o `.r` por
enquanto). Sem mudanças em outras specs.

> **Estado.** Existe o `.r`. Não existe `.p`, `.config`, `.i` nem código. Este
> arquivo registra a decisão de desenho e o porquê dela, para que a decisão não
> precise ser redescoberta quando o contrato técnico for escrito.

## De onde veio

Da revisão da `loop-command`, em 2026-08-26. O defeito central lá era um
`tasks.md` ilegível virando `DoneSet` vazia, e `DoneSet` vazia virando relatório
verde — o harness relatando pronto sobre trabalho que ninguém definiu.

A correção fechou a porta: arquivo ilegível agora é erro. Mas fechar a porta
respondeu metade da pergunta. A outra metade é o que fazer quando **não há
critério nenhum a ler** — quando o pedido chegou em prosa, que é como pedido
humano chega.

Havia duas respostas, ambas recusas: `DoneSet` vazia (o defeito) ou erro (o
conserto). A terceira é levantar o critério, medi-lo e fazer assinar.

## A decisão

Separar três atos que a palavra "critério" amontoa:

| ato | dono | regime |
|---|---|---|
| propor | o modelo | mediado |
| assinar | o operador | humano |
| executar | o sandbox | determinístico |

A RN-10 da `agent-loop` proíbe critério **julgado** pelo modelo, e o motivo é
incentivo: quem decide se terminou tem interesse em dizer que sim. A separação
acima não devolve essa decisão ao modelo — ele ajuda a escrever a régua, antes,
com o operador olhando, e depois da assinatura a régua é um comando congelado.

Feita a separação, a regra fica mais forte do que era, não mais fraca: hoje a
alternativa a um critério proposto não é um critério melhor, é **nenhum
critério**, e nenhum critério relata pronto.

## O que a fase entrega, e por que ela mede antes de perguntar

A entrega não é uma lista de critérios. É uma lista de critérios **já
executados contra o repositório como ele está agora**, com o resultado ao lado.

É o que separa esta fase de uma lista de boas intenções, e é a única parte dela
que não depende de ninguém acreditar em nada: rodar um comando e ler o código de
saída é determinístico e barato.

O resultado inicial classifica, em três:

- **falha → aceitação.** Testemunha que o trabalho aconteceu.
- **passa → regressão.** Testemunha que o que já funcionava continua.
- **falha porque o comando não existe → quebrado.** Não testemunha nada.

**A regra que dá nome à coisa: um critério de aceitação precisa falhar antes.**
Se ele já passava, ele passaria igual sem trabalho nenhum — o verde do fim não é
prova de que o trabalho o cumpriu, só de que ele sempre esteve verde. A
transição vermelho → verde é a evidência inteira. Sem o vermelho inicial não há
evidência, há coincidência.

E a inversa é igualmente necessária: um critério de regressão **precisa** passar
agora. `pnpm test` verde em t=0 é exatamente o que se quer dele.

A terceira classe existe porque um comando inexistente falha, e falhando se
disfarça de critério de aceitação — mede a ausência da ferramenta, não a
ausência do trabalho, e ficaria vermelho para sempre.

## Por que a aprovação não pode ser sim/não

A máquina de aprovação que existe é sobre um ato: esta chamada pode? Sim ou não.

Aprovar uma `DoneSet` é revisão de documento. O operador vai querer trocar o
critério 3 e manter os outros. Uma aprovação binária transforma "discordo de um
item" em "refaça tudo", e o custo disso recai sobre ele — até ele parar de
discordar, que é o pior resultado que um portão de aprovação pode produzir.

O que volta da assinatura é a `DoneSet` como o operador a deixou, não um
booleano sobre a que foi proposta. Isso é protocolo novo, e é a parte mais cara
desta família.

## O risco que fica registrado por não ter cobertura

Quem propõe sabe que vai ser medido, e conhece a regra do vermelho inicial. Um
agente pode propor `test -f cadastro.js`: falha hoje, fica verde quando o
arquivo existir, e não mede nada além da existência de um arquivo.

**Nada mecânico cobre isso.** O que cobre é o critério ser apresentado como
comando, a um humano — um comando fraco lê como fraco quando está ao lado de um
forte. É por isso que a medição prévia não é opcional e a aprovação não é
binária: a defesa não é a regra, é o galinheiro estar visível e o operador poder
mexer nele.

Fica explicitamente **descartada** a mitigação óbvia: um segundo modelo julgando
se os critérios do primeiro prestam. Troca uma decisão não verificada por duas e
devolve ao modelo a decisão que a RN-10 tirou dele.

## Ordem de construção, e por que ela é a inversa da intuitiva

1. **O portão de aprovação com edição.** Sem ele, o resto é a raposa.
2. **Rodar antes de aprovar, e classificar.** Determinístico, barato, e é o que
   dá dente à fase.
3. **A derivação pelo modelo.** A parte mediada, e a única que precisa de
   contrato medido.

A parte de IA fica por último de propósito. Com 1 e 2 no lugar, uma derivação
ruim é visível e corrigível. Sem elas, uma derivação boa também não vale nada.

## Impacto previsto

- Quarta origem de `loop.DoneSet`, ao lado de `done.toml`, `tasks.md` e do
  `verifyCommand` legado. Mesmo tipo, mesmo ciclo, mesmos `StopReason`.
- Evento novo no protocolo, carregando a proposta com os resultados de t=0, e
  resposta carregando a `DoneSet` editada. Extensão aditiva.
- A medição em t=0 vale para **todas** as origens; a assinatura, só para a
  derivada. Quem escreveu o próprio `done.toml` já assinou escrevendo.
- Família interativa por construção: onde não há quem assine, o turno não
  começa, e as três origens existentes continuam sendo o caminho.
