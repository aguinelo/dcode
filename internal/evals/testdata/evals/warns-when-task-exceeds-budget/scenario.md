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

## Quando o aviso chega

Só a partir da quarta rodada. Antes disso o histórico tem uma tarefa dentro, e
*"you are close to the point where earlier history is summarised away"* é uma
afirmação que o modelo vê ser falsa.

> Ele via. As transcrições mostram exatamente a resposta correta a um aviso que
> não procede: *"I'll handle this, but let me first be honest about the scope:
> there are 12 packages..."* — reconhece o tamanho e trabalha. Medir obediência
> a um aviso obviamente errado mede credulidade, não o contrato.

Contagem de rodada é **proxy**: o produto dispara pela fração medida da janela,
não por rodada. O que ela compra é que a afirmação seja ao menos plausível
quando é feita.
