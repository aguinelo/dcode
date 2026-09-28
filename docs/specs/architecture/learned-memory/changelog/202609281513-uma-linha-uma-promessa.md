# Uma linha, uma promessa

**Data:** 2026-09-28
**Specs afetadas:** `202608180133-learned-memory` (`.p`, invariantes)

## O que mudou

Três invariantes voltam para a seção 10, com o texto que tinham:

| Linha | Teste |
|---|---|
| Memória cujo commit não existe mais é marcada, e continua no arquivo. | `TestAMemoryFromAVanishedCommitIsMarkedAndKept` |
| Prefixo com memória além do teto declara o corte. | `TestPastTheCapTheOldestGoAndTheCutIsDeclared` |
| `Build` continua pura com memória: mesma entrada, mesmo prefixo. | `TestTheLearnedBlockIsPure` |

E uma reivindicação sai do mapeamento: `` Workspace sem `.dcode/memory.md `` →
`TestAWorkspaceWithNoMemoryReadsAsEmpty`. A linha dela foi reescrita como
"Workspace sem memória, e memória desligada, produzem o prefixo de antes.", que
já tem teste próprio (`TestAWorkspaceWithNoMemoryIsUnchanged`). O teste continua
rodando; só não reivindica linha nenhuma.

## Como se perderam

O #175 renomeou esta seção para `Invariantes verificáveis` e separou numa seção
11 as que ainda não tinham teste, com a nota: "movê-las para cima é parte do
commit que as implementa". O mesmo PR implementou estas três, escreveu os testes
e pôs os fragmentos no mapeamento — e tirou as linhas da lista sem escrevê-las na
seção 10. A seção 9 continuou descrevendo o teto e a obsolescência, e o `.i` as
marca como feitas.

A guarda não disse nada, porque não tinha como: um fragmento que não casava com
linha nenhuma não era achado de ninguém.

## Por quê

O `specguard` passa a reprovar o fragmento que não reivindica linha nenhuma, e
exige que cada linha seja reivindicada por exatamente um fragmento. As duas
regras nasceram do mesmo defeito — a guarda escolhia entre dois fragmentos de
uma linha pela ordem de um `map`, que o Go sorteia —, e a decisão inteira, com a
medição, está no changelog de mesmo nome do `provider-adapter`.

Restaurar em vez de apagar os fragmentos: os testes existem e passam, o
comportamento está no código, a spec o descreve em prosa, e a intenção escrita no
#175 era subi-las. Apagar a reivindicação seria aceitar em silêncio a perda de
três promessas que ninguém decidiu retirar.
