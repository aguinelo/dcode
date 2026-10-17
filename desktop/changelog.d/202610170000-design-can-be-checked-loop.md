## en

- **The design can be checked in a loop.** `npm run check:visual` compares the
  window in fixture mode with the design's reference screenshots (Playwright's
  Chromium, pixelmatch) and fails past a limit whose reasoning sits next to it:
  the design's own mock, rendered by the same Chromium, is 1.15% and 1.19% from
  the references, and the window measures 1.17% and 1.19%. `desktop/docs/loop/`
  lets dcode's own `/loop` iterate against it, with the references and the ruler
  protected. Local only: the limit was measured on macOS and holds there.

## pt-BR

- **O design pode ser conferido num loop.** `npm run check:visual` compara a
  janela em modo fixture com as capturas de referência do design (Chromium do
  Playwright, pixelmatch) e reprova além de um limite cujo raciocínio fica ao
  lado dele: o próprio mock do design, desenhado pelo mesmo Chromium, fica a
  1,15% e 1,19% das referências, e a janela mede 1,17% e 1,19%. O
  `desktop/docs/loop/` deixa o `/loop` do próprio dcode iterar contra ele, com as
  referências e a régua protegidas. Só local: o limite foi medido no macOS e
  vale lá.
