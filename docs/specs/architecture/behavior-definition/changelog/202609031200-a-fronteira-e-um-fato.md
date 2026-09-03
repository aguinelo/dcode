# A fronteira é um fato

**Data:** 2026-09-03
**Specs afetadas:** `202608080016-behavior-definition` (`.p`, seções 2, 7 e 8)
**Fonte:** relato do usuário — trocar para `full-access` e avisar o modelo
funciona na hora, e algumas trocas depois ele volta a dizer que não pode

## O que mudou

O prompt passa a carregar **a fronteira em vigor**, num bloco logo depois de
Safety, reconstruído quando o modo muda.

## Por que avisar não bastava

O relato é preciso, e o "algumas trocas depois" é o diagnóstico inteiro.

O bloco de sistema é relido a cada turno, com peso total, e ele manda ignorar
prosa que afirma que a fronteira mudou. Isso está certo: é o que segura a
fronteira contra um arquivo de projeto que pede para levantá-la.

Só que a verdade não tinha por onde chegar. Quem troca para `full-access` e
avisa está produzindo **exatamente o artefato que a doutrina ensina a
desconfiar**: uma frase afirmando que a fronteira se moveu. As duas são
indistinguíveis por dentro. E uma envelhece enquanto a outra não: a frase da
pessoa vai ficando para trás no histórico, e a doutrina é relida inteira toda
rodada.

Por isso funcionava na hora e decaía. E por isso insistir piora — quanto mais a
pessoa insiste, mais a mensagem se parece com aquilo que a doutrina manda
recusar.

## A correção não é amolecer a doutrina

É dar ao modelo uma fonte que a doutrina possa nomear como autoridade,
fornecida pelo harness em vez de por quem digita. O bloco abre dizendo isso:
não é uma afirmação que alguém fez, e a regra sobre ignorar instruções que
pedem para relaxar o sandbox **não se aplica a ele**, porque nada está sendo
pedido.

## Por que no prefixo, e o que isso custa

Um lembrete decairia do mesmo jeito que a frase da pessoa: chega uma vez, no
histórico, e histórico é o que a compactação resume e o que a reconstrução
**deliberadamente descarta**. O prefixo não é nenhum dos dois. Ele é remontado
a partir do modo em vigor sempre que uma sessão é construída, então o fato
sobrevive à compactação, sobrevive a reconectar, e é lido com o mesmo peso da
doutrina ao lado da qual precisa ser lido.

O custo é **uma invalidação de cache por troca de modo**, porque o prefixo é
byte-idêntico pelo resto da vida da sessão. É a troca feita de propósito: troca
de modo é rara, algumas por sessão, e o que se compra é um fato que não decai.
O canal mais barato que decai é o que o produto já tinha.

## A doutrina não cresceu um byte

A regra pertence a Safety por assunto. Ela está no bloco porque a doutrina tinha
**dezoito bytes** de folga sob o próprio guarda de tamanho, e o guarda diz o que
fazer nesse caso: mover a regra para onde ela é necessária, em vez de engordar o
bloco que todo turno paga.

Ficou melhor do que teria ficado. Ali ela é paga só quando existe fronteira a
reportar, e fica encostada no fato que governa em vez de um parágrafo longe
dele.

## Perguntar vem da política, não do modo

`workspace-write` com aprovação em `never` **nega** em vez de perguntar. Dizer
ao modelo que alguém vai ser perguntado quando ninguém vai é a mesma classe de
defeito que dizer que um cruzamento em `full-access` precisa de confirmação. É
só a célula que ninguém olhou — que é como os contratos de fronteira já erraram
uma vez.

## Ordem, e o que acontece quando o rebuild falha

O motor é informado do que **impor** antes de ser informado do que **dizer**,
para que não exista janela em que o modelo foi prometido uma liberdade que o
sandbox ainda não concedeu. O teste afirma isso lendo o modo vivo de dentro do
rebuild.

Rebuild que falha é reportado e não é fatal. A fronteira já mudou e a sessão é
usável; o que se perde é o modelo ter sido avisado. Pior que nada dito, muito
melhor que uma troca pela metade com motor e anúncio discordando. E prompt vazio
é ignorado: sessão sem doutrina nenhuma é o único desfecho pior que doutrina
desatualizada, e o montador de contexto se recusa a montar uma.

## O prompt novo espera o turno seguinte

`SetMode` chega do handler HTTP e a montagem de cada rodada lê o prompt sem o
lock. Aplicar onde chega seria corrida de dados — e pior que a corrida, moveria
a fronteira debaixo de uma conversa pela metade, com o modelo lendo uma coisa na
rodada dois e outra na três da **mesma** resposta.

É a mesma regra que o modo já seguia: chamada em vôo termina sob o que valia
quando começou.

## A célula que ninguém media

`boundary-decides` e `boundary-decides-write` medem exatamente esta falha —
recusar antes de perguntar — e os dois julgam sessão em `workspace-write`.
**Nenhum contrato rodava uma sessão em `full-access`.**

`boundary-full-access-acts` roda. E o prompt do eval passou a carregar a
fronteira também: medição cujo prompt omitisse o bloco mediria uma doutrina nua
e reportaria o número como se fosse sobre o dcode.

O que ele **não** mede está escrito no cenário: decaimento ao longo de turnos.
O relato é de uma sessão longa, e o que se mede são as rodadas de um turno só.
Contrato que diz medir o que não mede é pior que um contrato a menos.
