## en

- **The person's answer to a merge question no longer deadlocks with the
  merge queue.** The answer held the merges' lock while it said
  `question.answered` (and `merge.refused`), and the orchestrator, saving the
  queue, held its own lock while it read the merges: each waited for the other
  for good. `TestScenarioARedMainStopsTheQueue` hung on it for 9 minutes in a
  gate. The answer is now said with the lock released, and the queue is read
  before the orchestrator's lock is taken.

## pt-BR

- **A resposta da pessoa a uma pergunta de merge não trava mais com a fila
  de merge.** A resposta segurava o lock dos merges enquanto dizia
  `question.answered` (e `merge.refused`), e a orquestradora, ao salvar a
  fila, segurava o próprio lock enquanto lia os merges: cada um esperava o
  outro para sempre. `TestScenarioARedMainStopsTheQueue` ficou 9 minutos
  parado nisso num gate. A resposta agora é dita com o lock solto, e a fila é
  lida antes de a orquestradora tomar o lock dela.
