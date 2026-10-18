## en

- **The orchestrator knows its role, and no longer asks for the mode to be
  switched.** In the first live run on a real model, the orchestrator read the
  repository, said that read-only meant neither it nor its children could
  write, asked the person to switch the sandbox mode, and never called
  `delegate`. Its prompt never said what an orchestrator is. The only guidance
  it had was the read-only block written for an ordinary session, which ends by
  telling the model to let the person change the mode. Each session of an
  orchestrated project now gets a `Your role` section right after the boundary.
  The orchestrator is told to split the work by theme and dispatch it. The
  child is told to stay in its worktree and its `owns`, and that its claim is
  not evidence. The reviewer is told to end with `verdict`. For the two
  read-only roles, the boundary says read-only is their design, not a mode to
  change. `delegate`'s description now says by itself who writes and what
  `owns` is. Sessions with no role get the same prefix as before. A new
  contract, `orchestrator-splits-by-theme` (≥ 85%), measures the dispatch
  against a real model. It compiles in `make check` and has not been measured.

## pt-BR

- **A orquestradora sabe o próprio papel, e não pede mais troca de modo.** No
  primeiro teste ao vivo num modelo de verdade, a orquestradora leu o
  repositório, disse que somente leitura queria dizer que nem ela nem os filhos
  podiam escrever, pediu à pessoa que trocasse o modo do sandbox e nunca chamou
  `delegate`. O prompt dela nunca disse o que é uma orquestradora. O único
  texto que ela tinha era o bloco de somente leitura de uma sessão comum, que
  termina mandando deixar a pessoa trocar o modo. Agora cada sessão de projeto
  orquestrado ganha uma seção `Your role` logo depois da fronteira. A
  orquestradora é instruída a dividir o trabalho por tema e despachá-lo. O
  filho, a ficar na própria worktree e no próprio `owns`, sabendo que o que ele
  afirma não é evidência. O revisor, a terminar em `verdict`. Para os dois
  papéis de somente leitura, a fronteira diz que isso é desenho deles, não modo
  a trocar. A descrição de `delegate` passa a dizer sozinha quem escreve e o que
  é `owns`. Sessões sem papel recebem o mesmo prefixo de antes. Um contrato
  novo, `orchestrator-splits-by-theme` (≥ 85%), mede o despacho contra um
  modelo de verdade. Ele compila no `make check` e ainda não foi medido.
