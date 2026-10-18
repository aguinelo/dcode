## en

- **`dcode orchestrate` opens the project's orchestrator.** The new subcommand resolves the
  project of the folder through the daemon (`POST /v1/projects`, which finds the project another
  worktree or a link of the same tree already is) and opens the TUI on its orchestrator. It
  attaches to a running daemon and never starts one of its own: the orchestrator outlives the
  terminal, and an embedded daemon would end with it, taking the children along. With no daemon it
  says to run `dcode serve`; a daemon that conducts no projects is said naming
  `orchestrator.enabled` and `record.enabled`. `dcode` alone stays the ordinary conversation. On
  open, the list is read from `/children` and `/questions`, after the history, so the replay
  cannot bring back a child the daemon already let go. `^Q` opens the oldest question with the
  recommended answer under the cursor, the answers of merge and ceiling questions in the person's
  language; Enter answers through `POST /v1/sessions/{id}/questions/{qid}`, and an answer that lost
  to one given elsewhere (`question_already_answered`) is said. Under the orchestrator `^R` lists
  its children, and choosing one attaches to it, where its approval is answered; a child the
  daemon no longer holds is said, never continued by guess. Decision
  `client-tui/changelog/202610181200-dcode-orchestrate-abre-a-orquestradora.md`.

## pt-BR

- **`dcode orchestrate` abre a orquestradora do projeto.** O subcomando novo resolve o projeto da
  pasta pelo daemon (`POST /v1/projects`, que reconhece o projeto que outra worktree ou um link da
  mesma árvore já é) e abre a TUI na orquestradora dele. Ele anexa a um daemon que já roda e nunca
  sobe um próprio: a orquestradora sobrevive ao terminal, e um daemon embutido morreria com ele,
  levando os filhos. Sem daemon, diz para rodar `dcode serve`; daemon que não conduz projetos é
  dito nomeando `orchestrator.enabled` e `record.enabled`. `dcode` sozinho continua a conversa
  comum. Ao abrir, a lista é lida de `/children` e `/questions`, depois do histórico, para que a
  releitura não traga de volta filho que o daemon já soltou. `^Q` abre a pergunta mais antiga com a
  resposta recomendada sob o cursor, e as respostas das perguntas de merge e de teto no idioma da
  pessoa; Enter responde pelo `POST /v1/sessions/{id}/questions/{qid}`, e resposta que perdeu para
  outra dada em outro lugar (`question_already_answered`) é dita. Sob a orquestradora, `^R` lista
  os filhos, e escolher um anexa a ele, onde a aprovação dele é respondida; filho que o daemon não
  tem mais é dito, nunca continuado por palpite. Decisão
  `client-tui/changelog/202610181200-dcode-orchestrate-abre-a-orquestradora.md`.
