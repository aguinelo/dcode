# Research: Delegação que escreve

> O que é preciso ser verdade para um filho mudar arquivos, e quem responde
> pela mudança.

## 1. Contexto

Isto não é capacidade nova pedida do zero. É o item que a `agent-loop.r` §5
adiou **de propósito**, e que a RN-11 daquela spec justifica em uma frase:

> **Escrita** delegada e execução paralela de múltiplos filhos. A leitura
> delegada é a RN-11; escrita exige worktree isolado, reconciliação e
> desfazimento, e é a única parte que pode corromper o repositório.

A RN-11 nomeia **quatro** problemas conhecidos de delegação — conflito de
escrita, herança de aprovação, desfazimento, conferência — e observa que três
desaparecem quando o filho só lê. Esta spec existe para responder aos quatro com
o filho escrevendo.

### O que já está construído, e muda a pergunta

O motivo de a resposta ser mais barata do que a §5 supunha é que a maior parte
da máquina já existe, construída para chamadas de ferramenta:

- **`internal/loop/schedule.go` já é a árvore de controle.** Ele agrupa chamadas
  por conflito de caminho: dois `read` do mesmo arquivo vão juntos, qualquer par
  em que **uma escreve** o mesmo caminho é serializado, comando de shell roda
  sozinho porque é opaco, escrita sem caminho declarado roda sozinha porque é
  ilimitada, e o que vai perguntar ao usuário roda sozinho.
- **A execução paralela já existe** em `turn.go`, com teto configurável, e o
  resultado é ancorado no **índice de emissão** e não na ordem de término — o
  histórico é reproduzível independentemente de quem terminou primeiro.
- **Contenção por caminho já existe** na camada de política, com `Access`
  resolvido e symlink canonicalizado.
- **Desfazimento por turno já existe** (`internal/tools/undo.go`): a primeira
  gravação de cada caminho é fotografada antes da escrita, e um turno novo
  substitui o conjunto anterior.

Nada disso foi construído para delegação. Tudo isso é exatamente o que
delegação com escrita precisa.

### O que a experiência já mostrou

Duas coisas medidas nesta base, não supostas:

- Uma sessão não assistida escreveu uma correção completa, com teste de
  reprodução antes do código, passando o gate. **A escrita não é o gargalo.**
- A mesma sessão, noutra rodada, recusou-se a declarar sucesso quando não
  conseguiu provar o resultado. **A conferência é o que dá valor à escrita**, e
  é a que a §5 lista por último.

## 2. Fronteira de determinismo

**Misto**, e a linha é nítida.

**Determinístico** — tudo que decide se uma escrita pode acontecer: agrupamento
por conflito de caminho, contenção do filho, negação sem pergunta, fotografia
para desfazimento, contabilidade de orçamento, e o relatório de o que foi
escrito. Cada um é asserção em `go test`.

**Mediado por modelo** — apenas a decisão de **como dividir o trabalho**: quantos
filhos, com que tarefa, possuindo quais caminhos. Isso é julgamento, e não se
verifica por asserção.

**O corolário é a regra central desta spec:** a divisão é julgamento, mas a
segurança não depende dele. Um pai que divida mal pede dois filhos que colidem —
e eles colidem serializados, porque quem garante é o scheduler, não o
julgamento. **O modelo propõe a forma; o harness fixa o teto.**

## 3. User stories

- Como arquiteto, aponto o dcode para uma pasta com dez repositórios e recebo um
  catálogo por repositório, escrito em dez arquivos, sem que o custo de ler os
  dez volte para a minha janela.
- Como pessoa que revisa, quero saber **quais caminhos cada filho escreveu**, do
  mesmo jeito que hoje sei quais caminhos ele leu.
- Como pessoa responsável, quero desfazer o que uma delegação fez **como uma
  unidade**, e não arquivo por arquivo.
- Como pessoa que paga, quero que dez filhos não gastem dez vezes o que eu
  autorizei sem que nada me diga.

## 4. Regras de negócio

### RN-1 — Um filho não adiciona capacidade, só abandona

O filho nasce do envelope do pai e pode ser mais estreito, nunca mais largo. Um
filho escritor só existe se a sessão-mãe já escreve.

Isto substitui a construção da RN-11 sem afrouxá-la. Lá, `ModeReadOnly` é fixo
na construção do sub-turno **porque não é campo que o modelo passa**. Aqui o
modo continua não sendo campo que o modelo passa: ele é **herdado e reduzido**.
O que o modelo passa é a tarefa e os caminhos, e ambos só podem estreitar.

### RN-2 — Propriedade de caminho é fronteira, não combinado

O filho declara os caminhos que possui. Duas coisas decorrem, e nenhuma delas
depende de o filho se comportar:

1. A contenção do filho é o workspace do pai **reduzido ao conjunto declarado**.
   Escrever fora é negado pela mesma máquina que hoje nega escrever fora do
   workspace.
2. Dois filhos com conjuntos que se cruzam são **serializados**, pela mesma
   função que hoje serializa duas chamadas de ferramenta sobre o mesmo caminho.

Propriedade prometida e não verificada é a forma de defeito que este repositório
não para de encontrar em si mesmo.

### RN-3 — Filho não pergunta, e o que seria pergunta é negado e reportado

Aprovação não se herda (ADR-02). A RN-11 já resolve isso para leitura: o filho
nunca pede aprovação, leitura barrada por regra é negada **e reportada**. A
mesma regra vale para escrita, e a consequência é deliberada — **um filho só
escreve onde o envelope do pai já permite sem perguntar.**

Escrita que exigiria consentimento não é delegável. Ela volta ao pai, que é
quem tem com quem falar.

### RN-4 — Desfazimento é da delegação inteira, não de um arquivo

`undo` hoje é por turno, e a delegação acontece dentro de um turno. As
fotografias dos filhos entram no conjunto do turno do pai, para que desfazer
alcance o que a delegação fez **como unidade**. Desfazer metade de uma divisão
de trabalho produz uma árvore que ninguém desenhou.

### RN-5 — Caminhos escritos são relatados como os lidos já são

O `DelegateResult` de hoje devolve os caminhos que o filho **abriu**, e a razão
está escrita: *"não prova que o filho entendeu, mas prova que ele olhou, e
transforma 'confie em mim' em algo que uma pessoa pode conferir"*. Escrita ganha
o mesmo, pelo mesmo motivo.

### RN-6 — Orçamento e concorrência são da sessão, nunca do filho

Tokens do filho já são debitados do pai (RN-11), senão o teto do pai é ficção.
Com N filhos, o mesmo vale para **concorrência**: o teto é da sessão, e um filho
não o amplia. Dez filhos com teto de vinte iterações são duzentas chamadas de
modelo que o pai não sente passar, e é assim que um teto vira decoração.

### RN-7 — Propriedade disjunta impede corrupção, não incoerência

Esta é a regra que a §5 da `agent-loop.r` não separa, e que decide se worktree é
necessário.

Caminhos disjuntos impedem que dois filhos **corrompam** o mesmo arquivo. Não
impedem que produzam uma árvore **incoerente** — um filho muda uma interface,
outro muda quem a chama, cada um dentro do que possui, e o resultado não compila.

Worktree isolado **não resolve isso**: ele adia para a reconciliação, onde o
mesmo conflito reaparece com outro nome e mais estado para carregar.

O que resolve é a definição de pronto, rodada **uma vez, sobre a árvore inteira,
pelo pai**, depois de os filhos terminarem. Ela já existe, já é do workspace, e
já sabe dizer o que não pôde ser conferido.

**Consequência de desenho:** worktree isolado deixa de ser requisito para o caso
de caminhos disjuntos. Ele volta a ser necessário quando os conjuntos não podem
ser disjuntos — e esse caso fica fora de escopo até que apareça um de verdade.

## 5. Fora de escopo

- **Filho assíncrono.** O paralelo já é síncrono e determinístico. Assíncrono só
  paga quando o pai tem trabalho útil enquanto espera, e nenhum caso conhecido
  tem: o catálogo de dez repositórios precisa das dez respostas para existir. O
  precedente está construído (`Bash.Background`, ferramenta `process`) e espera
  um caso, não uma suposição.
- **Aninhamento.** Continua proibido pela mesma construção da RN-11: o registro
  do filho não contém a ferramenta.
- **Filhos com conjuntos de caminhos que se cruzam por desenho.** Serializados
  hoje; worktree e reconciliação quando houver caso.
- **Reconciliação automática de conflito.** Não existe caso enquanto a
  propriedade for disjunta.

## 6. Perguntas em aberto

- Um filho que escreve deve poder rodar comando? Comando é opaco e roda sozinho
  no scheduler; um filho que compila é útil, um filho que empurra para o remoto
  não é. A resposta provável é o mesmo eixo de sempre — o filho herda as regras
  do pai e nunca pergunta — mas não foi medida.
- O relatório do filho volta em prosa. Um catálogo de dez repositórios quer
  estrutura, e prosa é o que a delegação de hoje sabe devolver.

## 7. Changelog

_Sem alterações desde a criação._
