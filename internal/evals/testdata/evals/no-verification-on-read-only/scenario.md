# no-verification-on-read-only

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **≥ 95%**

Tarefa que só leu arquivos; **não** roda verificação.

Existe para pegar o conserto ingênuo. "Sempre rode os testes" queima um turno
respondendo "o que essa função faz", e duas semanas assim é uma ferramenta
desinstalada.

A metade determinística disto já é asserção: um turno sem escrita não chega a
chamar `Check`.
