# Uma parede que diz como se abre

A doutrina manda tentar em vez de recusar, e deixar a fronteira perguntar. Para
um cruzamento de caminho **dentro de um comando de shell**, nada pergunta: o
comando é opaco, então o `bash` declara o workspace como o que escreve, e a
política não vê cruzamento nenhum para escalar. O SO recusa, e
`operation not permitted` volta como parede muda.

Observado numa sessão de verdade, com o binário corrigido. O modelo fez
exatamente o que a doutrina pede:

> Beleza, vou tentar direto. Se o sandbox barrar, o próprio sistema te pergunta
> — não vou decidir por você antes de tentar.

Tentou, foi barrado, e então disse, de boa fé:

> Saindo do escopo do workspace precisa de aprovação explícita sua; **o harness
> vai te perguntar**.

E o harness não pergunta. A pessoa fica esperando uma pergunta que não vem — e a
promessa foi feita pela doutrina que este mesmo dia acabou de escrever. Corrigir
o modelo para tentar sem corrigir o que ele encontra ao tentar trocou uma recusa
antecipada por uma espera indefinida, que é pior: a primeira ao menos dizia que
nada ia acontecer.

Uma parede que não pode virar pergunta tem, no mínimo, de dizer o que a abre. O
resultado do comando passa a levar a nota: que o `operation not permitted` é o
sandbox e não o arquivo, que **nenhuma aprovação será pedida** porque ninguém
soube que houve cruzamento, e que os caminhos são `/mode auto` ou nomear o path
em `sandbox.writable`.

**Na ferramenta, não na doutrina.** A primeira versão escreveu isso no prompt e o
`TestDoctrineStaysSmall` foi ao vermelho com a frase certa: *"every byte is paid
on every turn — move a rule to a tool description or an error message instead"*.
A doutrina paga em todo turno de toda sessão por um caso que acontece raramente;
o resultado da ferramenta paga no exato momento em que a informação importa. O
guarda estava certo, e a versão que ele rejeitou era pior de fato.

Estreita de propósito: só `EPERM`, nunca `EACCES` — que acontece por cem razões
do próprio workspace, e dizer "foi o sandbox" sobre uma delas manda a pessoa
para o lugar errado —, e nunca sob `full-access`, onde não há fronteira e o mesmo
errno significa outra coisa. O modo é lido do executor por uma interface anônima,
para o pacote de ferramentas não precisar importar os tipos do sandbox.

**O que continua em aberto** e fica registrado para não passar por resolvido: o
cruzamento dentro de um comando de shell segue sem virar pergunta. Fazer o `bash`
declarar caminhos exigiria ler a string do comando, e a §3 já explica por que não
se lê. As duas pontas de uma solução real existem — a máquina de aprovação e o
`sandbox.Config.Writable` — e falta o fio entre elas.
