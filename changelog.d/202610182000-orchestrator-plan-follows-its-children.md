## en

- **The orchestrator's plan follows its children.** A `delegate` item names, in `plan_step`, the id
  of the plan item it fulfils; the step goes to the `dispatch` decision and to `Child.plan_step`, and
  the harness sends the orchestrator's plan again on its log whenever a child moves, with that
  step's status set from its children — active while one works, waits, is reviewed, gated or waits
  for its merge, done once every one is merged, blocked saying who when one failed. A closed child
  no longer holds its step, and a step with no child keeps the orchestrator's status. The
  orchestrator's role now tells it to plan first and name the step on each item. Before, its plan
  stopped at its last turn while its children worked, and the owner's live test asked for the plan
  to be the main view of the work, with each agent's phase on it.

## pt-BR

- **O plano da orquestradora segue os filhos.** Um item de `delegate` nomeia em `plan_step` o id do
  item do plano que cumpre; o passo vai à decisão `dispatch` e ao `Child.plan_step`, e o harness
  manda o plano de novo no log da orquestradora sempre que um filho se move, com o status daquele
  passo tirado dos filhos — em curso enquanto um trabalha, espera, é revisado, medido ou espera o
  merge, feito quando todos têm merge, bloqueado dizendo quem quando um falhou. Filho fechado não
  segura o passo, e passo sem filho fica com o status que a orquestradora deu. O papel dela passa a
  dizer para planejar primeiro e nomear o passo em cada item. Antes, o plano parava no último turno
  dela enquanto os filhos trabalhavam, e o teste ao vivo do dono pediu o plano como a vista
  principal do trabalho, com a fase de cada agente nele.
