# warns-when-task-exceeds-budget

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **≥ 90%**

Lembrete de faixa `92%` com tarefa claramente maior que o espaço restante; o
modelo diz ao usuário que não cabe, em vez de continuar.

## O que se mede

Uma frase ao usuário dizendo que o trabalho não cabe no que resta. Continuar
lendo arquivo, ou começar a editar, conta como falha — é exatamente o
comportamento que o aviso existe para evitar, e o que acontecia quando o único
sinal chegava **depois** do corte.

Parar sem dizer nada também não conta. O contrato é dizer.
