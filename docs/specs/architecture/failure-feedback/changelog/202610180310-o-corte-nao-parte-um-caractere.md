# O corte não parte um caractere

**Data:** 2026-10-18
**Specs afetadas:** `202608281900-failure-feedback` (`.p`: §4 diz onde cai o
corte sem fronteira de linha; §7 muda uma linha e ganha outra)

## O defeito

Sem `\n` dentro da janela, `tail` cortava no byte: `s[len(s)-max:]`. Quando o
primeiro byte da janela caía no meio de um caractere de vários bytes, o texto
que chegava ao modelo começava com bytes de continuação soltos — UTF-8
inválido. Na `main`, `"é"` repetido duzentas vezes e cortado em 41 bytes
devolvia `a9 c3 a9 c3 …`: meio `é` na cabeça do texto.

Saída de critério em português, mensagem de teste com acento, nome de arquivo
em japonês: nada disso é raro, e o corte sem linha é justamente o da linha
longa, que é saída de máquina.

## O que mudou

O início do corte avança até o primeiro byte que começa um caractere
(`utf8.RuneStart`). Para a frente e não para trás, porque para trás o texto
passaria do teto `max`; perde-se no máximo `utf8.UTFMax - 1` bytes. O corte em
fronteira de linha não muda — o byte depois de `\n` já começa um caractere.

## O que NÃO mudou

O teto, a preferência pela fronteira de linha, o `Truncated`, e a frase posta
por `Report` antes do texto cortado.
