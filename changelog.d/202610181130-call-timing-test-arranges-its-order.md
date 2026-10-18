## en

- **The test that a tool event carries when its call ran no longer depends on
  timing.** It bet on two sleeps — 40 ms against 1 ms — to make two calls
  overlap and the second finish first. Under a loaded machine the fast call's
  goroutine was once scheduled after the slow one had finished, and the test
  reported a sequence where the engine had done nothing wrong (1 failure in
  1200 runs with the whole suite as load). The order is now arranged with
  channels — the slow call returns only after the fast call's completion is
  emitted — and the engine's injectable clock ticks once per reading, so every
  "before" is decided by the order of the readings. It also checks that the
  fast call finished inside the slow one's interval, which catches an event
  that carries the start twice; the old version caught that only when the
  goroutines happened to start in emission order.

## pt-BR

- **O teste de que o evento de uma ferramenta diz quando a chamada rodou não
  depende mais de tempo.** Ele apostava em dois sleeps — 40 ms contra 1 ms —
  para que as duas chamadas se sobrepusessem e a segunda terminasse primeiro.
  Com a máquina carregada, a goroutine da chamada rápida chegou a ser agendada
  depois de a lenta já ter terminado, e o teste acusou uma sequência onde o
  motor não tinha errado nada (1 falha em 1200 execuções com a suíte inteira
  como carga). A ordem agora é montada com canais — a chamada lenta só retorna
  depois de emitida a conclusão da rápida — e o relógio injetável do motor
  avança um tique por leitura, de modo que todo "antes" é decidido pela ordem
  das leituras. O teste passa também a conferir que a chamada rápida terminou
  dentro do intervalo da lenta, o que pega um evento que carregue o início duas
  vezes; a versão antiga só pegava isso quando as goroutines por acaso
  começavam na ordem de emissão.
