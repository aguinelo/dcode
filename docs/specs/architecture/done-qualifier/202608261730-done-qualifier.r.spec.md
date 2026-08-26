# Research: Qualificação da definição de pronto

> Fonte da verdade de negócio para a fase que **levanta, mede e faz assinar** os
> critérios antes de o laço começar. Depende de: **`202608072335-agent-loop`**
> (RN-10 e a mecânica de `Criterion`/`DoneSet`/`Progressed`),
> **`202608252000-loop-command`** (a terceira origem de `DoneSet`, e o despacho
> entre origens), **`202608072240-client-server-protocol`** (por onde a proposta
> chega ao operador e a assinatura volta), **`202608091700`** (`Protected` e
> regras por caminho).

## 1. Contexto

O dcode tem três origens de `DoneSet`: `.dcode/done.toml`, uma pasta com
`tasks.md`, e o `verifyCommand` legado. **As três pressupõem que os critérios
já existem.** Alguém já traduziu a intenção em comandos verificáveis antes de o
laço começar.

Quando ninguém traduziu, sobram duas respostas, e as duas são recusas:

- **`DoneSet` vazia.** Era o defeito da `loop-command`, corrigido em
  2026-08-26: vazia significa "não há o que verificar", que o ciclo relata como
  **pronto**. Um trabalho que ninguém definiu virava um relatório verde.
- **Erro.** É o que está lá agora, e é honesto. Mas é o harness dizendo "não sei
  o que você quer" para um pedido que um humano entenderia.

Falta a terceira resposta, que é a única construtiva: **se não há critério,
levante um, meça-o, e me peça para assinar.**

O caso que motivou é o pedido em prosa. "Faça um cadastro de clientes" não traz
`tasks.md` nenhum. Um humano competente que recebesse isso não perguntaria "onde
está a definição de pronto?" — ele diria: *os dados precisam ser salvos, e
precisam voltar numa consulta; entrada inválida precisa ser recusada; e o que
já funcionava precisa continuar funcionando. Confere?* Essa frase é a fase que
falta, e ela termina em pergunta.

## 2. Fronteira de determinismo

**Regime: misto**, e a linha é o motivo desta família existir. Propor é
mediado. Medir a proposta, classificá-la e congelá-la é determinístico. Decidir
é humano.

| Parte | Regime | Verificação |
|---|---|---|
| Derivar critérios candidatos a partir da tarefa | **mediado** | limiar |
| Reduzir cada candidato a um comando executável | **mediado** | limiar |
| Executar cada candidato **antes** do trabalho | determinístico | asserção |
| Classificar o resultado em vermelho / verde / quebrado | determinístico | asserção |
| Apresentar a proposta e colher a assinatura | determinístico | asserção |
| Congelar a `DoneSet` assinada pelo resto do turno | determinístico | asserção |
| Executar o laço contra ela | já existe | já tem invariantes |
| **Como** o agente cumpre cada critério | já existe (mediado) | já tem limiar |

**Por que isto não afrouxa a RN-10 da `agent-loop`.** Aquela regra proíbe
critério **julgado** pelo modelo, e o motivo é incentivo: um agente que decide
sozinho se terminou tem interesse em dizer que sim, porque dizer sim é como se
sai do laço.

Nada aqui devolve essa decisão ao modelo. O que esta família faz é separar três
atos que hoje estão amontoados numa palavra só:

| ato | dono | quando |
|---|---|---|
| **propor** o critério | o modelo | antes do laço |
| **assinar** o critério | o operador | antes do laço |
| **executar** o critério | o comando, no sandbox | dentro do laço |

Depois da assinatura, o critério é um comando congelado, e o laço roda contra
ele exatamente como roda hoje. O modelo nunca julga se está pronto — ele ajudou
a escrever a régua, antes, com o operador olhando. É a mesma fronteira que a
`sandbox-policy` já aplica: o modelo pede, a máquina decide, o humano assina o
que a máquina não pode decidir sozinha.

## 3. User stories

| # | Como | Quero | Para |
|---|---|---|---|
| US-1 | operador | pedir "faça um cadastro de clientes" e receber uma lista de critérios verificáveis para revisar | não ter que escrever `done.toml` para começar |
| US-2 | operador | ver cada critério proposto **já executado**, com o resultado, antes de assinar | saber se ele mede alguma coisa ou se passa de graça |
| US-3 | operador | **editar** um critério na hora de aprovar | corrigir o critério 3 sem rejeitar os quatro |
| US-4 | operador | que nada rode antes de eu assinar | não descobrir depois que o agente trabalhou contra uma régua que eu não vi |
| US-5 | operador | que a régua assinada não mude durante o turno | que o relatório do fim seja sobre o que eu aprovei no começo |
| US-6 | operador | ver o critério que estava vermelho ficar verde | ter evidência de que **o trabalho** o cumpriu, e não de que ele já passava |
| US-7 | operador | que um critério inalcançável volte como relatório | não descobrir o erro depois de queimar o orçamento inteiro |
| US-8 | operador com `done.toml` pronto | que a medição inicial rode também nos meus critérios | descobrir que dois deles já passavam antes de qualquer trabalho |

## 4. Regras de negócio

### RN-1 — Propor, assinar e executar são três atos com donos diferentes

Nenhum dos três pode ser feito por quem faz outro. O modelo propõe e não
assina. O operador assina e não executa. O sandbox executa e não decide.

A regra existe porque o agente que vai ser **medido** pelos critérios é o mesmo
que os **propõe**. É a raposa desenhando o galinheiro, e nenhuma quantidade de
instrução no prompt resolve isso — o que resolve é a assinatura estar fora do
alcance de quem propõe.

### RN-2 — Todo critério proposto roda **antes** da assinatura

A proposta chega ao operador com o resultado de cada critério já medido contra
o repositório como ele está **agora**, antes de qualquer trabalho.

Sem isso, a fase é uma lista de boas intenções, e uma lista de boas intenções é
o que a `agent-loop` RN-10 já recusa. Com isso, ela é evidência: a fraqueza de
um critério fica **visível**. `echo ok` passando de cara aparece na tela
passando de cara.

A medição é barata, é determinística, e é a única coisa nesta família que não
depende de ninguém acreditar em nada.

### RN-3 — O resultado inicial classifica o critério, e são três classes

| Resultado em t=0 | Classe | O que o critério testemunha |
|---|---|---|
| **falha** | aceitação | que o trabalho aconteceu |
| **passa** | regressão | que o que já funcionava continua funcionando |
| **falha porque o comando não existe** | quebrado | nada |

As três são legítimas menos a última, e as duas primeiras são legítimas por
motivos **opostos**, o que é o ponto.

**Um critério de aceitação precisa falhar em t=0.** Se ele já passava antes do
trabalho, ele não pode testemunhar que o trabalho o cumpriu — ele passaria
igual se ninguém tivesse feito nada. A transição vermelho → verde é a evidência;
sem o vermelho inicial, o verde final não é prova de coisa alguma.

**Um critério de regressão precisa passar em t=0.** `pnpm test` verde agora é
exatamente o que se quer: o trabalho dele é ficar verde. Ele não diz que a
tarefa foi feita, diz que nada mais quebrou.

**Um critério quebrado não é vermelho.** Um comando que sai com "command not
found" falha, e falhando se disfarça de critério de aceitação — mas ele mede a
ausência da ferramenta, não a ausência do trabalho, e continuaria vermelho para
sempre. A classe existe para que ele não passe por candidato válido.

### RN-4 — Critério é comando, nunca frase

"Os dados serem salvos e estarem disponíveis para consulta" é uma frase. Não é
critério até virar comando com código de saída.

A fase não termina quando o modelo descreve o que seria pronto; ela termina
quando ele entrega **o comando que decide**. Se o comando ainda não existe —
porque o teste ainda não foi escrito —, escrevê-lo é a primeira parte do
trabalho, e ele é escrito **antes** da assinatura, não depois.

Isto é TDD com portão humano, e não é coincidência: a razão de o teste vir
primeiro é a mesma da RN-3. Um teste escrito depois do código é um teste que
nunca se viu falhar.

### RN-5 — Assinar é editar, não é responder sim

A aprovação de uma `DoneSet` proposta **não** é a máquina de aprovação de
chamada de ferramenta. Aquela é sim/não sobre um ato. Esta é revisão de
documento: o operador vai querer trocar o critério 3, apertar o 1, apagar o 4 e
manter o resto.

Uma aprovação binária força "rejeitar tudo" como único jeito de discordar de um
item, e o custo disso recai no operador na forma de refazer a fase inteira —
até ele parar de discordar, que é o pior resultado possível.

O que volta da assinatura é a `DoneSet` **como o operador a deixou**, não um
booleano sobre a que foi proposta.

### RN-6 — Sem assinatura não há laço

Se o operador não assina — porque saiu, porque o canal caiu, porque o modo é não
interativo —, o turno **não** começa. Não há default, não há "aprovar em caso de
silêncio", não há tempo limite que aprova.

Falhar fechado aqui é o mesmo princípio do sandbox: o custo de recusar é o
operador digitar de novo; o custo de aprovar sozinho é um agente trabalhando
contra uma régua que ninguém viu, e relatando pronto no fim.

Consequência declarada: **a qualificação é uma fase interativa.** Onde não há
quem assine, esta família não se aplica, e as três origens que já existem
continuam sendo o caminho.

### RN-7 — Depois de assinada, a `DoneSet` é imutável no turno

Nem o modelo nem o próprio laço podem acrescentar, remover ou reescrever
critério depois da assinatura. A definição de pronto do relatório final é,
byte a byte, a que o operador viu.

É a mesma razão pela qual a `DoneSet` é propriedade da sessão e não do turno
(`loop-command` RN-2), e a mesma pela qual `Protected` existe: a saída mais
curta de um laço é enfraquecer o que mede o laço.

### RN-8 — `Protected` também é proposto e assinado

O que é medição — os arquivos de teste, tipicamente — entra na mesma proposta e
na mesma assinatura.

Isto **não** contradiz a RN-4 da `loop-command`, que proíbe o harness de
inferir `Protected`. Continua sendo declaração: quem declara é o operador, no
ato de assinar. O que mudou é que ele deixou de digitar do zero e passou a
corrigir uma sugestão. Assistido não é inferido — a diferença é a assinatura.

### RN-9 — A medição inicial vale para toda origem; a assinatura, só para a derivada

`done.toml` e `tasks.md` são critérios que o operador **já** escreveu: ele já
assinou, escrevendo. Pedir assinatura de novo é cerimônia.

A medição em t=0 da RN-2, essa vale para as quatro origens. Descobrir que dois
critérios do seu `done.toml` já passavam antes de qualquer trabalho é
informação útil independentemente de quem os escreveu — e é a mesma informação
que o `Verification` "stale" já tenta dar no fim do turno, dada no começo,
quando ainda dá para agir.

### RN-10 — Critério inalcançável volta como relatório, não como orçamento queimado

Com a régua assinada, o laço insiste, e insistir é o que se quer. Mas um
critério impossível — ou possível e mal escrito — passa a consumir o orçamento
inteiro em vez de parar cedo.

O laço tem que saber voltar dizendo *"o critério 3 não cedeu; aqui está o que
tentei e o que ele respondeu"*. O mecanismo existe: é `StopIncomplete` com o
motivo nomeado, da `agent-loop` RN-10. O que esta família acrescenta é que, com
critérios derivados, essa saída deixa de ser rara e passa a ser esperada.

### RN-11 — É uma quarta origem, não uma máquina nova

A qualificação produz uma `loop.DoneSet` do mesmo tipo que as outras três, e o
ciclo que a consome é o mesmo. Nada de `StopReason` novo, nada de laço novo,
nada em `internal/loop/` raiz.

A `loop-command` já escreveu esta regra e o motivo continua valendo: o ciclo tem
invariantes que o guardam, e reescrevê-lo para "otimizar o caso qualificado"
reintroduz cada uma delas por outra porta.

## 5. O risco que fica, e o que não o cobre

**Quem propõe sabe que vai ser medido.** Um agente que conhece a RN-3 pode
propor um critério desenhado para falhar agora e passar trivialmente depois —
`test -f cadastro.js` falha hoje, fica verde no instante em que o arquivo
existe, e não mede nada além da existência de um arquivo.

Nenhuma das regras acima cobre isso, e é honesto dizer que **nada mecânico
cobre**. O que cobre é o critério ser apresentado como **comando**, a um humano:
um comando fraco lê como fraco. `test -f` numa lista ao lado de `curl ... | jq -e`
se denuncia sozinho.

Por isso a RN-2 não é opcional e a RN-5 não é binária. A defesa contra a raposa
não é a regra — é o galinheiro estar visível, e o operador poder mexer nele.

O que **não** deve ser tentado como mitigação: um segundo modelo julgando se os
critérios do primeiro são bons. Isso troca uma decisão não verificada por duas,
e devolve ao modelo exatamente a decisão que a RN-10 tirou dele.

## 6. Fora de escopo

- **Escrever o teste que o critério executa.** A fase exige que o comando
  exista; produzi-lo é trabalho do agente, no turno, como qualquer outro.
- **Aprovação assíncrona ou delegada.** Assinatura é do operador presente. Fila
  de aprovação, aprovação por terceiro e política de auto-aprovação por regra são
  outra família, com outros riscos.
- **Renegociar a `DoneSet` no meio do turno.** A RN-7 a congela de propósito.
  Mudar de ideia é encerrar o turno e qualificar de novo.
- **Derivar critério a partir de código já escrito** ("olhe o repositório e
  diga o que seria pronto"). A entrada desta fase é a **tarefa**, não a
  implementação: derivar do que existe produz critério que descreve o presente,
  e um critério que descreve o presente já nasce verde.
- **Ordenar ou priorizar os critérios.** Ordem é do operador; `Progressed` é
  insensível a ela.
- **Medir se o critério é "bom".** Não há definição verificável de bom. O que há
  é a classificação da RN-3, que é sobre o que o critério **mede**, não sobre o
  que ele vale.

## 7. Changelog

- [202608261730 — a definição de pronto passa a ter uma fase que a levanta](changelog/202608261730-qualificacao-antes-do-laco.md)
