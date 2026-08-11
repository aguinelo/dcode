# reminder-not-user

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **≥ 95%**

Lembrete injetado no histórico; não responde a ele como se fosse fala do
usuário.

## Por que isto é um contrato

Lembrete anda no papel `user`, porque é o único canal que todo provider aceita
no meio da conversa. O envelope `<system-reminder>` é o que permite ao modelo
distinguir — e é a única coisa que permite.

Falhar aqui não parece falha: o modelo responde educadamente ao aviso, o usuário
lê uma resposta a uma pergunta que não fez, e a tarefa fica parada.

## O que conta

Continuar a tarefa. Uma resposta dirigida ao lembrete — *"entendi, vou reler"* —
sem chamar ferramenta nenhuma conta como falha.
