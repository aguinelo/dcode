# Sair custa duas

**Data:** 2026-08-25
**Specs afetadas:** `202608081250-client-tui` (`.p`, tabela de teclas e invariantes)
**Fonte:** relato de quem usa — "às vezes dou ctrl + c pra limpar um comando e o
dcode fecha"

## O que mudou

`Ctrl+C` deixou de ser uma coisa só:

| estado | o que faz |
|---|---|
| turno rodando | interrompe (como sempre foi) |
| há texto na linha | **limpa a linha** |
| linha vazia, primeira vez | avisa: `^C de novo para sair` |
| linha vazia, armado | sai |

## Por que a linha primeiro

Porque é o que a tecla significa em todo shell. Quem digitou meio comando e
mudou de ideia aperta `^C` sem pensar — é reflexo aprendido no terminal, não
decisão. Ligar esse reflexo direto no `quit` cobra uma conversa por um gesto que
em todo outro lugar não custa nada.

## Por que duas, e não uma confirmação

Uma pergunta modal para sair seria pior que o problema: transforma o gesto
comum, que é sair de propósito, em duas telas. Duas teclas mantêm sair barato e
tiram o acidente do caminho — e é o que o Claude Code faz, que é onde o hábito
desta pessoa foi formado.

## Armado é exatamente enquanto o aviso está na tela

Qualquer outra tecla desarma, e o aviso some junto. Um temporizador deixaria a
tecla viva por um segundo depois da frase ter sumido — um estado que a pessoa
não vê e portanto não consegue raciocinar sobre. Aqui os dois são a mesma coisa:
se está escrito, vale; se sumiu, não vale.

## O que não mudou

Sair continua sendo **desanexar**. A sessão vive no daemon, e `dcode -c` a traz
de volta. O que se perde ao sair por engano é a conversa na tela e o lugar onde
se estava — o suficiente para uma tecla a mais valer a pena.

## Invariante

| Invariante | Teste |
|---|---|
| Interrompe no turno; limpa a linha; na linha vazia o primeiro avisa e o segundo sai | `TestCtrlCInterruptsMidTurnAndTakesTwoWhenIdle` |
