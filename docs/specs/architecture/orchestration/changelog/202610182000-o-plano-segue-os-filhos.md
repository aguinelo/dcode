# O plano da orquestradora segue os filhos

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §5.2, §10, §13); `behavior-definition` `.p`
§2.1, pela seção do papel da orquestradora.

## O que aconteceu

No teste ao vivo de 2026-10-18 o dono disse que o plano fixo como checklist (desktop D57) deixou
muito claro o que estava acontecendo, e pediu que ele fique como a vista principal da orquestradora,
com os agentes em paralelo: em que fase o trabalho está e como cada agente vai. Até aqui o plano da
orquestradora só mudava quando ela chamava `plan`, e ela fica `idle` enquanto os filhos trabalham —
o plano parava no último turno dela.

## O que mudou

- **Protocolo.** `Child` ganha `plan_step` (`omitempty`): o id do item do plano que o filho cumpre.
- **Ferramentas.** O item de `delegate` ganha `plan_step`; negativo recusa o lote inteiro, com
  `invalid_input`. `children` diz o passo de cada filho.
- **Núcleo.** `plan.go`: o harness lê o `plan.updated` da sessão da orquestradora, guarda o plano em
  `State.Plan`, e a cada `child.started` e `child.state` manda o plano com o status dos passos que
  têm filhos tirado deles. A linha `dispatch` leva `plan_step`.
- **Papel.** A seção da orquestradora diz para chamar `plan` antes de despachar, um item por tema,
  e nomear o passo em cada item de `delegate`; enquanto um item tem filhos, o status é do harness.

## O que o código decide onde o dono deixou a forma em aberto

- **Projeção, não decisão.** O status do passo é função dos estados dos filhos, e cada mudança de
  estado já tem a sua linha no log. Uma linha a mais por passo diria duas vezes a mesma coisa.
- **A regra do status.** Algum filho falhou: `blocked`, com quem e o fim do turno (`incomplete`
  além de `max_returns`). Todos com merge: `done`. Senão `active`. Fechado não segura o passo — a
  orquestradora fechou de propósito, e o passo volta ao status que ela deu. Vários passos podem
  ficar `active` juntos: é o paralelo que o dono pediu ver, e a regra de um ativo só é da
  ferramenta `plan`, para o que a orquestradora escreve.
- **O plano da orquestradora é guardado cru.** O harness guarda o que `plan` escreveu e manda a
  projeção; o que ele mandou volta pelo mesmo log e é reconhecido pela igualdade com o último
  mandado, sem virar o plano dela. Assim um filho fechado devolve o passo ao status que ela deu, e
  não ao que o harness tinha posto.
- **Passo que o plano não tem não recusa o lote.** O `plan.updated` chega ao harness por uma
  assinatura assíncrona do log; recusar um `delegate` por um plano que ainda não chegou seria uma
  corrida. O filho aparece fora do plano, no desktop e na lista.
