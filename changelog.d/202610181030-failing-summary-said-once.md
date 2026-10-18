## en

- **A failing summary is said once, not every iteration.** Since a failed
  summary stopped compacting, the history stayed above the automatic trigger's
  threshold, so every later iteration of the turn built the same plan, asked
  for the same summary and failed on the same cause. A revoked credential or a
  spent quota cost one more provider call and one more `compaction_failed` per
  round. The trigger now latches its first failure for the rest of the turn,
  and that one warning says automatic compaction will not try again in it. A
  `context_size` refusal still forces a compaction, and the next turn tries
  again. Decision `agent-loop/changelog/202610181030-resumo-que-falha-e-dito-uma-vez.md`.

## pt-BR

- **Resumo que falha é dito uma vez, não a cada iteração.** Desde que resumo
  que falha deixou de compactar, o histórico ficava acima do limiar do gatilho
  automático, e cada iteração seguinte do turno montava o mesmo plano, pedia o
  mesmo resumo e falhava pela mesma causa. Credencial revogada ou cota esgotada
  custavam uma chamada a mais ao provider e um `compaction_failed` a mais por
  rodada. O gatilho agora trava na primeira falha até o fim do turno, e esse
  único aviso diz que a compactação automática não tenta de novo nele. A
  recusa `context_size` ainda força a compactação, e o turno seguinte volta a
  tentar. Decisão `agent-loop/changelog/202610181030-resumo-que-falha-e-dito-uma-vez.md`.
