## en

- **The TUI's `/loop` asks the daemon to run it.** The terminal no longer
  conducts its own loop: it sends what was typed to `POST /v1/loops`, follows
  the run's stream and says what changed — the plan, the session the loop moved
  into, each proposal, the end — in the words it always used. Typing the same
  `/loop` again after a proposal goes on with the loop waiting for that review;
  a different one is refused, naming the loop that waits. Interrupting the
  loop's turn now stops the loop instead of skipping to the next folder, and a
  message queued on a loop's session waits for the loop instead of starting a
  turn in the folder the daemon is leaving; it goes on its own only when the
  loop finishes, and after a stop it stays until ↵ sends it. The
  run carries what was typed, what its survey found in each folder and what
  each proposal wrote (`args`, `criteria`, `unmet`, `unavailable`, `error`,
  `proposal_criteria`, `proposal_summary`), so no client measures twice. Two
  clients now see one loop; the TUI keeps no copy of the sequence or the
  grammar.

## pt-BR

- **O `/loop` da TUI pede ao daemon que o rode.** O terminal deixa de
  conduzir o próprio loop: manda o que foi digitado para `POST /v1/loops`,
  segue o fluxo do loop e diz o que mudou — o plano, a sessão para onde o loop
  foi, cada proposta, o fim — com as palavras de sempre. Digitar o mesmo
  `/loop` de novo depois de uma proposta segue o loop que espera aquela
  revisão; um diferente é recusado, nomeando o loop que espera. Interromper o
  turno do loop agora para o loop em vez de pular para a próxima pasta, e uma
  mensagem enfileirada na sessão de um loop espera o loop em vez de abrir um
  turno na pasta que o daemon está deixando; ela sai sozinha só quando o loop
  termina, e depois de uma parada fica até ↵ mandá-la. O loop
  carrega o que foi digitado, o que a pergunta achou de cada pasta e o que cada
  proposta escreveu (`args`, `criteria`, `unmet`, `unavailable`, `error`,
  `proposal_criteria`, `proposal_summary`), para nenhum cliente medir duas
  vezes. Dois clientes agora veem um loop só; a TUI não guarda cópia da
  sequência nem da gramática.
