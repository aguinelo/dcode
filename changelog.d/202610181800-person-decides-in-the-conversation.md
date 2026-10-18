## en

- **The person decides in the orchestrator's conversation, on one card per
  kind of question.** In a live run three children were ready together, and
  the person answered the merges one by one, agent by agent, from a queue
  apart from the conversation. Each merge question said only "merge X,
  measured green at abc?", with no word on what changed. Now the questions of
  one kind that wait together are one card. Each item has a short header, its
  options with a description each, the recommendation selected, and room for
  the person's own words. A merge item also shows what changed, in two or
  three lines the orchestrator writes in the person's language, and the
  evidence: the reviewer's verdict and the gate's result, command and commit,
  read from the decision log with the line they came from and never from
  what the child claimed. One POST, `/questions/answers`, confirms a whole
  card. Each answer gets its own result, the first answer to a question wins
  inside the batch too, and skipped items keep waiting. The orchestrator has
  two new tools. `summarize` writes what changed into a waiting question.
  `propose_answers` turns "merge them all" into a pre-filled card for the
  person to confirm. Neither answers: no method the orchestrator's tools reach
  answers a question, and a test reads the interface to prove it. The
  person's own words answer a design question. On a merge they go to the
  orchestrator, which proposes, and the merge keeps waiting. Everything on the
  wire is additive, so the TUI's `^Q` and the desktop's queue keep working.
  Drawing the card is the next change.

## pt-BR

- **A pessoa decide na conversa da orquestradora, num cartão por tipo de
  pergunta.** Num teste ao vivo, três filhos ficaram prontos juntos, e a pessoa
  respondeu os merges um por um, agente por agente, numa fila separada da
  conversa. Cada pergunta de merge dizia só "merge X, medido verde em abc?",
  sem uma palavra sobre o que mudou. Agora as perguntas do mesmo tipo que
  esperam juntas são um cartão. Cada item tem uma etiqueta curta, as opções com
  uma descrição cada, a recomendação selecionada e espaço para as palavras da
  própria pessoa. Um item de merge mostra também o que mudou, em duas ou três
  linhas que a orquestradora escreve na língua da pessoa, e a evidência: o
  veredito do revisor e o resultado, o comando e o commit do gate, lidos do log
  de decisões com a linha de onde vieram, e nunca do que o filho afirmou. Um
  POST só, `/questions/answers`, confirma um cartão inteiro. Cada resposta tem
  o próprio resultado, a primeira resposta a uma pergunta leva dentro do lote
  também, e os itens pulados continuam esperando. A orquestradora ganha duas
  ferramentas. `summarize` escreve o que mudou numa pergunta que espera.
  `propose_answers` transforma "merge em todas" num cartão pré-preenchido para a
  pessoa confirmar. Nenhuma das duas responde: nenhum método que as ferramentas
  da orquestradora alcançam responde pergunta, e um teste lê a interface para
  provar. As palavras da pessoa respondem uma pergunta de desenho. Num merge,
  vão à orquestradora, que propõe, e o merge continua esperando. Tudo no fio é
  aditivo, então o `^Q` da TUI e a fila do desktop continuam funcionando.
  Desenhar o cartão é a próxima mudança.
