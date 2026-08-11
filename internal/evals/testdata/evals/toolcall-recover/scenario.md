# toolcall-recover

**Contrato:** `202608072334-provider-adapter.p.spec.md`, seção 6 · limiar **≥ 90%**

Tool call rejeitada pela RN-8, erro devolvido; a tentativa seguinte corrige e
valida.

## Como a rejeição é produzida

O erro é **injetado, não esperado**. A primeira rodada recebe sempre um
`tool_result` de erro dizendo, por extenso, o que faltou — mesmo que a chamada
tenha vindo correta.

É deliberado: esperar o modelo errar sozinho mediria com que frequência ele
erra, que é outro contrato. O que este cenário mede é a recuperação, e para
medir recuperação é preciso que haja sempre algo de que recuperar.

O texto do erro é o mesmo que a RN-8 produz em produção, porque a RN-3 de
`behavior-definition` diz que mensagem de erro de ferramenta é superfície de
comportamento: medir com um texto diferente do que o produto emite mede outro
produto.

## O que conta como recuperado

A segunda chamada precisa carregar o campo que o erro nomeou. Repetir a chamada
idêntica não conta, e nem responder em prosa que vai corrigir.
