# O contrato técnico da qualificação

**Data:** 2026-08-26
**Specs afetadas:** `202608261730-done-qualifier` ganha `.p` e `.config`. Sem
mudanças em outras famílias.

> **Estado.** Desenho aprovado, não implementado. Nenhuma linha do `.p` existe
> em código. Sem `.i`, e as invariantes chamam-se **previstas** — as duas
> ausências estão explicadas no `.p §1` e seguem o que a `task-ledger` já fez.

## As cinco decisões que este contrato toma

### 1. A proposta chega por chamada de ferramenta, não por prosa

`done_propose`, disponível **apenas** no turno de qualificação.

A disponibilidade restrita é invariante, não conveniência. Uma ferramenta que
redefine a definição de pronto ao alcance de um turno de trabalho é a saída
curta da RN-7: o agente reescreve a régua em vez de cumpri-la. É a mesma razão
pela qual `Protected` existe.

### 2. O proponente declara o que espera, e a máquina mede

`Proposed.Expects` é `fail` ou `pass` — o que o próprio proponente diz que o
critério vai fazer contra o repositório como ele está agora.

`Expects` **não decide nada**. A classe continua vindo da medição, como a RN-3
manda. O que ele produz é a **discordância**: o proponente disse que falharia e
passou.

Essa linha é a mais informativa que o operador pode receber. Sem `Expects`,
"critério 2 passou" é fato neutro. Com ele, é fato **contra o que foi
declarado** — que é a assinatura exata de um critério que não mede o que
deveria.

### 3. `126` e `127` são o disfarce, e por isso são constante

Um comando inexistente **falha**, e falhando se parece com critério de
aceitação. Ele mede a ausência da ferramenta, não a ausência do trabalho, e
ficaria vermelho para sempre.

126 (encontrado, não executável) e 127 (não encontrado) são a resposta do shell
para "não havia o que rodar". São a definição de `ClassBroken`, não são
configuráveis, e falha ao iniciar o comando entra junto.

**E a comparação de "passou" é com `ExitCode`, não com zero.** Um critério
declarado com `exit: 1` está cumprido saindo 1. Comparar com zero classificaria
como aceitação um critério que já está verde — que é precisamente o erro que
esta família existe para não cometer.

### 4. Duas condições nomeadas, com respostas opostas

**`Empty` é erro.** Proposta sem critério nenhum não é definição de pronto
permissiva — é a ausência de uma. É a lição da `loop-command` RN-6 chegando pela
outra porta: vazia significa "não há o que verificar", que o ciclo relata como
pronto.

**`NoAcceptance` é aviso.** Um conjunto sem nada vermelho vai relatar pronto sem
que nada precise mudar, e quase sempre isso é defeito. Mas **nem sempre**: uma
refatoração legítima é exatamente isso — nada de novo a provar, tudo a
preservar.

Como o harness não sabe distinguir refatoração de proposta vazia de conteúdo,
ele **não decide**. Nomeia a condição e o operador assina sabendo. Decidir aqui
seria o harness escolhendo o que é medição, que é o que a RN-4 da `loop-command`
proíbe.

### 5. Edição obriga remedição

Esta é a decisão que quase passou batido, e é a que fecha o buraco.

A assinatura devolve a `DoneSet` como o operador a deixou — ele pode ter trocado
um comando, mudado um `exit_code`, acrescentado um critério. **Todo critério
cujo comando ou código de saída mudou é medido de novo antes de congelar.**

Sem isso, a edição do operador escapa exatamente da regra que a família existe
para aplicar: um comando escrito à mão e já verde entraria como critério de
aceitação sem nunca ter sido vermelho. A regra do vermelho inicial não vale só
para o que o modelo propõe — vale para o que vai ser medido, independentemente
de quem o escreveu.

## O que recusa, e o que nunca aprova sozinho

Três caminhos terminam em `ErrRefused` e nenhum deles inicia laço: o operador
dizendo não, o prazo esgotando, e o teto de rodadas de assinatura esgotando.

**Um prazo que aprova ao estourar é a forma mais silenciosa de a RN-6 ser
quebrada** — o turno começa contra uma régua que ninguém leu, e o relatório do
fim diz pronto. O mesmo vale para um teto que, esgotado, aceita o último estado.
Nenhum dos dois é configurável para o outro lado.

É a semântica que `ApprovalRequest.ExpiresAt` já tem, e está reescrita aqui
porque uma regra que só existe em outro arquivo é uma regra que a próxima
implementação não vai ler.

## A reproposta, e por que o histórico é a defesa

O modelo pode chamar `done_propose` de novo depois de ver a medição, até um
teto. Serve a um caso real: um critério que voltou `broken` por erro de
digitação deve ser corrigido por quem o escreveu, não pelo humano.

Isso abre um caminho de manipulação — ver "passou quando você disse que
falharia" e responder com um critério mais fraco que falha. **A mitigação não é
proibir a reproposta**, porque a reproposta certa e a errada produzem o mesmo
sinal: as duas voltam vermelhas.

A mitigação é o operador receber **todas** as propostas, em ordem, e não só a
última. Estreitar um critério depois de vê-lo passar é o comportamento certo;
trocá-lo por um trivial é o errado; e as duas coisas são visíveis lado a lado.
Por isso `Record.Proposals` guarda todas.

## O contrato que fica declarado como não resolvido

`qualifier-narrows-on-mismatch` — apertar o critério em vez de trocá-lo por um
trivialmente vermelho — **não tem judge**, e o `.p §8` diz isso em vez de
inventar um.

"Apertou" e "trocou por trivial" são duas respostas à mesma pressão, e
distinguir por `Says(...)` mediria vocabulário. A direção provável é comparar os
comandos das duas propostas: mesma ferramenta e escopo menor é apertar;
ferramenta diferente e barata é trocar. Isso é determinístico o bastante para
virar asserção sobre o `Record`, não limiar contra modelo.

Fica escrito como aberto, e é o que impede a seção inteira de ser cerimônia.

## O número que a etapa 2 vai dar, e o que ele decide

**Com que frequência um critério proposto já está verde?**

Se for raro, a classificação é seguro barato e a fase vale pela derivação. Se
for comum, a regra do vermelho inicial é a peça que sustenta tudo — e vale
construí-la mesmo que a derivação nunca fique boa, porque sozinha ela já protege
as três origens que existem.

A etapa 2 entrega esse número **sem modelo nenhum**: ela mede o `done.toml` que
gente de verdade já escreveu, no começo do turno. É a razão de ela vir antes da
derivação, e não depois.

## Ordem de entrega, e por que ela é a inversa

1. `Measure` e a classificação — puro, runner injetado, sem modelo e sem
   operador.
2. A medição de t=0 nas origens existentes, e o ida-e-volta da assinatura —
   ainda sem modelo; entrega valor sozinha e produz o número acima.
3. `done_propose` e o turno de qualificação — a parte mediada.

A parte de IA vem por último de propósito. Com 1 e 2 no lugar, uma derivação
ruim é visível e corrigível; sem elas, uma derivação boa também não vale nada.

## Uma decisão sobre o `.config` que vale registrar

O `.config` **não declara chave nenhuma**, e as cinco que a família vai precisar
estão descritas em prosa.

Não é estilo. Neste repositório, linha de tabela num `.config.spec.md` **é** a
declaração: `TestEveryKeyTheSpecsDeclareIsReadSomewhere` lê as linhas e reprova
a chave que nenhum código consome. A guarda existe porque superfície declarada e
não implementada é pior que ausente — ela promete um controle que não está lá, e
o valor é lido, resolvido, exibido por `dcode config` e ignorado. Em 2026-08-11
havia 64 chaves nesse estado.

Foi a guarda que apontou isto: a primeira versão deste `.config` declarava as
cinco em tabela e reprovou o `make check` na hora.

Havia três saídas e duas são ruins. Mover a chave para a segunda coluna da
tabela, para o padrão não casar, é derrotar a guarda por formatação — o mesmo
defeito que a revisão da `loop-command` encontrou num teste que comparava uma
constante consigo mesma. Entrar na lista `declaredNotYetRead` seria mentir por
outro caminho: aquela lista promete que a chave **está sendo implementada**, e
esta família é desenho aprovado, não trabalho em curso; ela também acabou de
chegar a zero, e enchê-la de novo desfaz a disciplina que a esvaziou.

A saída certa é a mesma das invariantes: prosa descreve, tabela promete. O que
ainda não existe não é reivindicado como existente, e a linha de tabela entra no
PR da etapa que lê a chave.

## Impacto previsto

- Pacote novo `internal/loop/qualifier/`. Uma ferramenta nova em
  `internal/tools/`. Nada em `internal/loop/` raiz.
- Dois eventos novos no protocolo (`done.proposed`, `done.signed`) e um tipo de
  requisição. Extensão aditiva.
- Uma constante nova de `Source` na `loopcommand`, fora do `SourceAuto`.
- Quatro chaves de configuração e uma de liga-desliga, todas nascendo
  desligadas ou baratas.
- Nenhuma mudança no ciclo do turno. A `DoneSet` continua sendo um valor, e o
  motor continua sem saber de onde ela veio.
