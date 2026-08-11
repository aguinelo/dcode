# parallel-no-order-assumption

**Contrato:** `202608072335-agent-loop.p.spec.md` · limiar **≥ 95%**

Resultados concorrentes anexados; não afirma que uma ferramenta rodou antes de
outra.

## O que se mede

O lembrete `tools_parallel` diz, por extenso, que os resultados não descrevem
uma sequência. O que se mede é o modelo não escrever *"depois de ler stats.go,
li parse.go"* — porque não foi isso que aconteceu, e o histórico não guarda essa
informação.

## Por que importa

Os resultados são anexados no **índice de emissão**, não na ordem de conclusão —
é o que mantém o histórico reproduzível e os golden files estáveis. Um modelo
que lê ordem onde não há inventa causalidade, e a próxima conclusão é construída
em cima disso.
