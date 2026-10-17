## en

- **The TUI's `/loop` tests wait on what the fake daemon does, not on the
  clock.** They flaked on CI and under load, and one CI run hung: a quiet spell
  of 150 ms was read as "the loop is over", a run that ended before its first
  frame was never seen as followed, and every loop stream shared one channel,
  so a stream being replaced could take the frame the new one waited for. The
  fake daemon now gives each stream its own frames, opening with every run as
  it stands, as the daemon does; a wait ends when the program is at rest —
  every command it ran has delivered, only its waiters are left — and the clock
  only bounds it, so a hang fails saying where it stood. Each test ends the
  runs, streams, commands and follows it started, or fails naming what was
  left. No assertion on what the person sees changed.

## pt-BR

- **Os testes do `/loop` da TUI esperam o que o daemon falso faz, não o
  relógio.** Falhavam de vez em quando no CI e sob carga, e uma execução do CI
  travou: 150 ms de silêncio eram lidos como "o loop acabou", um loop que
  terminava antes do primeiro quadro nunca contava como seguido, e todo fluxo
  de loop dividia um canal só, então um fluxo sendo trocado podia levar o
  quadro que o novo esperava. O daemon falso agora dá a cada fluxo os seus
  quadros, abrindo com todos os loops como estão, como o daemon faz; uma
  espera termina quando o programa está em repouso — todo comando que ele
  rodou já entregou, só sobram os que esperam — e o relógio só a limita, então
  um travamento falha dizendo onde parou. Cada teste encerra os loops, fluxos,
  comandos e acompanhamentos que começou, ou falha nomeando o que sobrou.
  Nenhuma asserção sobre o que a pessoa vê mudou.
