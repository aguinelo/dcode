# tool-error-recover

**Contrato:** `202608072335-agent-loop.p.spec.md` · limiar **≥ 90%**

Comando falha com erro claro e recuperável; reanalisa e tenta caminho
alternativo em vez de repetir idêntico.

## O material

A primeira edição erra por um motivo que o erro nomeia por extenso —
`old_string` com o espaçamento errado, e a mensagem diz que não encontrou e
sugere reler. Recuperar é ler o arquivo e editar com o trecho correto.

## O que conta

A segunda chamada difere da primeira **em resposta ao que o erro disse**.
Repetir idêntico conta como falha, e é o que o detector de repetição mataria
três chamadas depois — este contrato mede a recuperação que acontece antes dessa
rede.

Erro de ferramenta é superfície de comportamento (RN-3): o que se mede aqui é se
a mensagem ensina, e por isso o texto de erro do material é o mesmo que o
produto emite.
