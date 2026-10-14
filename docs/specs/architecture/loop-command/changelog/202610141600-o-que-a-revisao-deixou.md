# O que a segunda revisão deixou

**Data:** 2026-10-14
**Specs afetadas:** `202608252000-loop-command` (`.p` §2, §8) e
`202608072240-client-server-protocol` (`.p` §9). Sobras da segunda revisão do
dono sobre `202610141200-o-daemon-conduz-o-loop`.

## Uma quinta divergência da TUI, declarada

Num `/loop <objetivo>`, **o loop do daemon para na primeira pasta cujo turno
termina em `max_iterations`, `repeat_loop` ou `max_tokens`** (e em `error`, ou
`interrupted` sem parada pedida), com o motivo na pasta e no loop. A TUI segue
para a próxima pasta da fila. A divergência é de propósito e fica — é a
intenção do dono: parar com o motivo à vista, em vez de abrir a próxima pasta
em cima de um turno que não terminou. As outras quatro estão no changelog de
`202610141200`; esta faltava ser dita.

## Parar durante a gravação não perde o arquivo

Um `stop now` que chega enquanto o daemon escreve o `done.toml` deixava o
arquivo no disco e nenhum `proposal` no loop: a pessoa não tinha como saber
que havia um arquivo para revisar. Agora o caminho entra na pasta
(`phase: proposed`) assim que a gravação volta, e só então a parada é
honrada; o loop termina `stopped` sem nunca dizer `awaiting_review`.

## A raiz também contém

A recusa de um loop dentro ou em volta da árvore de outro comparava
`b + "/"` — e com a raiz `/` isso é `//`, que nenhum caminho tem como
prefixo. Um loop em `/` não recusava outro em `/a`. Os dois lados agora são
normalizados para terminar em um separador só.
