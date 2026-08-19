# O filho diz o que escreveu

PR 2 da §9 do `.p`. Três coisas, e as três respondem à mesma pergunta: **o que
esse trabalho dividido de fato produziu?**

## `Wrote` viaja com a conclusão

Pelo mesmo motivo que `Read` já viajava, e o comentário original já dizia: não
prova que o filho entendeu, prova que ele olhou, e transforma "confie em mim" em
algo que uma pessoa confere por amostragem.

Com trabalho dividido há uma pergunta a mais, e ela vem antes de qualquer outra:
**qual filho mexeu no quê.** O relatório passa a responder.

## O desfazimento alcança o que foi delegado

`undo` é por turno, e a delegação acontece dentro de um. Sem isto o desfazimento
do pai alcançaria tudo menos justamente a parte que ele delegou — e desfazer
metade de uma divisão de trabalho deixa uma árvore que ninguém desenhou.

`State.Adopt` move três coisas, cada uma pelo seu motivo:

- **as fotografias**, para haver o que repor. Onde os dois têm, vence a **do
  pai**: a primeira do turno é de onde o turno partiu, e a do filho, mais tarde,
  registra um estado que o próprio turno produziu.
- **os registros de arquivo**, porque o `undo` recusa arquivo que mudou depois
  que o turno o deixou, e esse julgamento precisa do hash do que foi realmente
  escrito. Aqui vence a **do filho**, que escreveu por último e é com quem o
  disco concorda.
- **o conjunto de escritos**, porque "esta sessão mudou alguma coisa" é o fato
  que a definição de pronto lê, e trabalho feito por filho continua sendo
  trabalho feito.

## Filho que não respondeu é nomeado

Com vários em voo, N−1 respostas é o caso **comum**, não o excepcional. A falha
passa a carregar a tarefa do filho, para que ele seja identificável entre os
irmãos.

Devolver N−1 calado é resultado incompleto com cara de completo — a forma de
defeito que este repositório não para de encontrar em si mesmo, e a mesma que a
delegação somente-leitura já evitava com `could not read`.

## O que falta

Quatro invariantes seguem previstas, no PR 3: filho nunca pede aprovação, teto
de concorrência da sessão, tokens debitados do pai, e a definição de pronto não
rodando dentro de um filho. As três últimas já são verdade no código — herdadas
da delegação somente-leitura — e o que falta é o teste que as reivindique com o
filho escrevendo.
