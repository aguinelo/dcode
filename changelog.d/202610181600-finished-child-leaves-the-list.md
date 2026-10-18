## en

- **A merged child leaves the orchestrator's list after the retention.**
  `/children` dropped only a closed child once `orchestrator.closed_retention`
  had passed since it ended; a merged one stayed for good, and since a merged
  child cannot be closed, nothing could ever take it off. Both terminal states
  now leave the same way — the decision log keeps their history, as Q-E says.
  A failed child is not terminal and still stays until the person closes it.

## pt-BR

- **Um filho merged sai da lista da orquestradora depois da retenção.**
  `/children` tirava só o fechado, passado `orchestrator.closed_retention`
  desde o fim; o merged ficava para sempre, e como um merged não se fecha,
  nada o tirava dali. Os dois terminais agora saem do mesmo jeito — o log de
  decisões guarda o histórico deles, como diz a Q-E. Um filho `failed` não é
  terminal e continua na lista até a pessoa fechá-lo.
