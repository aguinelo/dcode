## en

- **The daemon ruler reads the orchestrator's agent list.** `check:daemon`
  gains `orchestra-lists` (`tests/daemon/orchestra.mjs`), the scenario step 14
  of the orchestration plan asked for. It runs a real `dcode serve` with the
  orchestrator on and a scripted model that answers each role from its own
  script. The window opens the orchestrator through "◆ orquestradora" in
  "Nova", and two children are listed under it, in the Agentes tab and with ↳
  in Sessões. One child shows "esperando você" with its approval. The other's
  merge is answered from the queue, it merges locally, and it leaves the list
  after a one-second retention. The waiting one leaves too once the person
  closes it. Each claim is checked against `/children`, `/questions` and git,
  not only against what is drawn. No attribute was added to the window: the
  scenario reads the ones D64 and D66 already put there. The owner authorized
  this addition to the protected ruler.

## pt-BR

- **A régua do daemon lê a lista de agentes da orquestradora.** O
  `check:daemon` ganha o `orchestra-lists` (`tests/daemon/orchestra.mjs`), o
  cenário que o passo 14 do plano da orquestração pedia. Ele roda um
  `dcode serve` real com a orquestradora ligada e um modelo roteirizado que
  responde a cada papel com o roteiro dele. A janela abre a orquestradora pelo
  "◆ orquestradora" do "Nova", e dois filhos aparecem sob ela, na aba Agentes
  e com ↳ em Sessões. Um filho mostra "esperando você" com a aprovação que
  pede. O merge do outro é respondido pela fila, ele faz o merge local e sai
  da lista depois de uma retenção de um segundo. O que espera também sai,
  depois que a pessoa o fecha. Cada afirmação é conferida contra `/children`,
  `/questions` e o git, não só contra o que foi desenhado. Nenhum atributo
  entrou na janela: o cenário lê os que D64 e D66 já puseram. O dono
  autorizou este acréscimo à régua protegida.
