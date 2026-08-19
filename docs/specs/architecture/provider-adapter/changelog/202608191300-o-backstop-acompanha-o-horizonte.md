# O backstop acompanha o horizonte do modelo

`MiniMaxM3.DefaultLimits()` passa de 200 para 2.000 iterações.

## O que aconteceu

A justificativa do número já estava escrita, e já dizia outra coisa. O
comentário no código e a §2.1 do `.p` citam, os dois, uma execução da MiniMax
com **1.959 tool calls** como a razão de M3 não usar o teto cauteloso de 50 — e
então fixam o teto em **200**, um décimo da execução citada.

Um teto justificado por uma evidência e escrito abaixo dela é um teto que a
evidência não sustenta. Ninguém tinha esbarrado nele, então a distância entre o
motivo e o número nunca foi cobrada.

## O que cobrou

Uma sessão não assistida neste repositório, corrigindo a guarda de nome não
usado (`internal/specguard/unused_test.go`). Ela escreveu o teste de reprodução
primeiro, separou declaração de leitura, restaurou as duas desculpas com o
motivo de cada uma, e o `make check` passou com 95,1% — a correção estava certa
e completa.

E ela terminou em `max_iterations`, na rodada 200, sem conseguir dizer que tinha
acabado. **O trabalho sobreviveu; a resposta não.** Quem lesse só o resultado
veria uma execução que estourou o teto, que é como uma execução malsucedida se
parece.

## O que continua valendo

O teto nunca foi o mecanismo principal contra loop patológico, e continua não
sendo: isso é o detector de repetição em 3 chamadas idênticas. O teto é
backstop, e um backstop existe para o caso em que todo o resto falhou — sizá-lo
abaixo do horizonte do modelo faz dele um limite de trabalho, não uma rede.

`claude` fica em 50 e `Generic` em 50. Os dois números foram dimensionados por
outra coisa — um refactor cruzando dez arquivos, e a cautela devida a uma
família desconhecida — e nenhum deles foi contrariado por medição.

## O que este changelog não resolve

Estourar o teto continua sendo uma surpresa: o modelo não sabe quantas rodadas
lhe restam, então não tem como priorizar fechar. É o item 1 do `ROADMAP.md`, e
subir o teto o adia sem responder. Dez vezes mais espaço torna o encontro mais
raro, não mais legível.
