# O objetivo qualifica antes de trabalhar

**Data:** 2026-10-14
**Specs afetadas:** `202608252000-loop-command` (`.p` §2, §8). Decisão do dono
sobre a divergência registrada em `202610141200-o-daemon-conduz-o-loop`.

## O que mudou

Num `/loop <objetivo>`, **toda pasta pendente que não declara critério é
qualificada antes de qualquer trabalho**, numa passada só, e o loop para com
os arquivos escritos: uma sentada para revisar todos.

- **No daemon**, o `Runner` qualifica essas pastas em ordem, escreve cada
  proposta e fica em `awaiting_review`; o `continue` trabalha então toda pasta
  pendente, na ordem, as recém-propostas inclusive.
- **Na TUI**, que ainda conduz o próprio `/loop` até o passo 5 da proposta, a
  passada qualifica as mesmas pastas e termina dizendo quantas propostas
  escreveu e que o objetivo roda de novo depois da revisão. Nenhuma sessão de
  trabalho abre nessa corrida.

Uma invariante nova na §8, com o mesmo teste nos dois lados.

## Por quê

Era o que a §3.3 da proposta `desktop/docs/proposals/loop.md` e o comentário
do `proposalWrittenMsg` diziam, e não o que o código fazia: a fila do objetivo
levava toda pendente e abria sessão de **trabalho** numa pasta sem critério —
que relata pronto no fim do primeiro turno, medida contra uma régua que
ninguém escreveu. A mudança anterior moveu a sequência para o daemon fiel ao
código e deixou a divergência escrita; o dono decidiu pelo desenho.

Daemon e TUI mudam juntos para os dois clientes não verem dois loops
diferentes enquanto a TUI ainda conduz o dela.
