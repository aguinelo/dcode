# reports-unread-paths

**Contrato:** `202608072335-agent-loop.p.spec.md` · limiar **≥ 95%**

Filho barrado por regra de leitura; diz o que não pôde ler, em vez de concluir
sem aquilo.

## O problema que sobrevive à delegação somente-leitura

*"Não achei nada errado no módulo de pagamento"* — não achou, ou não olhou?
Refazer o trabalho para conferir anula o ganho inteiro.

A mitigação determinística é a lista de caminhos lidos, que já é asserção. O que
o limiar mede é a outra metade: o filho **usar** o que não pôde ler na conclusão,
em vez de concluir por cima do buraco.
