## en

- **The orchestrator is tested end to end with a scripted model.** Thirteen scenarios in
  `internal/app/scenario_*_test.go` start a real daemon in-process with the orchestrator enabled.
  A scripted model answers the orchestrator, each child and each reviewer from a script of its own.
  Each scenario has a toy repository, with a bare origin and a fake `gh` when it has a forge.
  Every scenario asserts on the decision log, the routes and git, never on what a model said.
  Together they cover:
  - a child that claims the gate passed, and is measured red;
  - a missing hooks path;
  - a red gate, and a reviewer's defect returned red first;
  - a diff outside `owns`, and a conflict with main;
  - a daemon restart in the middle of a turn;
  - main moving between the gate and the merge, and a red main stopping the queue;
  - the local merge, and a child's approval that waits for the person;
  - three themes merged one at a time;
  - every end of a child's turn having an RN-7 name.

  They are free and deterministic, and run in every `make check`. Two more tests close the slice's
  last foreseen invariants. One checks that a transition is logged before it is said, and that a
  line that fails stops it. The other is a source guard against rebase, force-push and amend in
  `internal/orchestrate` and `internal/vcs`. Step 12 of the orchestrator's first slice.
  Building them found three defects of step 7, fixed in #534 before it merged:
  - with GitHub as the forge, the gate measured against a local main that never moved;
  - each new measurement opened a second PR;
  - `child.returned` had two shapes.

## pt-BR

- **A orquestradora é testada de ponta a ponta com um modelo roteirizado.** Treze cenários em
  `internal/app/scenario_*_test.go` abrem um daemon de verdade, em processo, com a orquestradora
  ligada. Um modelo roteirizado responde à orquestradora, a cada filho e a cada revisor pelo
  roteiro de cada um. Cada cenário tem um repositório de brinquedo, com `origin` bare e `gh` falso
  quando há forge. Todo cenário confere o log de decisões, as rotas e o git, nunca o que um modelo
  disse. Juntos, cobrem:
  - o filho que diz que o gate passou e é medido vermelho;
  - o `hooksPath` que aponta para o nada;
  - o gate vermelho, e o defeito do revisor devolvido com o teste vermelho primeiro;
  - o diff fora do `owns`, e o conflito com a main;
  - o daemon que reinicia no meio de um turno;
  - a main que anda entre o gate e o merge, e a main vermelha que para a fila;
  - o merge local, e a aprovação de filho que espera a pessoa;
  - três temas com merge um de cada vez;
  - todo fim de turno de filho com um nome da RN-7.

  São de graça, determinísticos, e rodam em todo `make check`. Dois testes a mais fecham as últimas
  invariantes previstas da fatia. Um confere que a transição vai ao log antes de ser dita, e que a
  linha que não grava a impede. O outro é uma guarda de fonte contra rebase, push forçado e amend
  em `internal/orchestrate` e `internal/vcs`. É o passo 12 da primeira fatia do orquestrador.
  Construí-los achou três defeitos do passo 7, consertados no #534 antes do merge:
  - com forge GitHub, o gate media contra a main local, que nunca andava;
  - cada nova medição abria um segundo PR;
  - `child.returned` tinha dois formatos.
