## en

- **The visual check stays out of CI for the reason that holds.** The documents
  kept it local because the reference images were not in the repository; they
  have been since the design handoff landed. Run once on the workflow's
  `ubuntu-latest`, it scores 1.76% and 1.79% against the 1.25% limit, identical
  on four runners: Linux's Chromium hints glyphs and antialiases them in LCD
  subpixels, where macOS — on which the limit was measured — and the references
  do neither. The limit stays: loosened to fit Linux, it would leave macOS more
  than 9,000 pixels of slack, room for a missing element to pass. `AGENTS.md`,
  the workflow, the ruler's note and the root changelog stop saying the
  references are missing, and `AGENTS.md` gives the reason that holds, with the
  launch flags that brought Linux to 1.19% and 1.20% — a ruler change, left for
  a decision of its own.

## pt-BR

- **A verificação visual fica fora do CI pelo motivo que vale.** Os documentos a
  deixavam local porque as imagens de referência não estavam no repositório;
  estão desde que o handoff do design entrou. Rodada uma vez no `ubuntu-latest`
  do workflow, ela mede 1,76% e 1,79% contra o limite de 1,25%, igual em quatro
  runners: o Chromium do Linux aplica hinting nos glifos e os suaviza em
  subpixels LCD, e nem o macOS — onde o limite foi medido — nem as referências
  fazem isso. O limite fica: afrouxado para caber o Linux, deixaria ao macOS
  mais de 9 mil pixels de folga, espaço para passar um elemento faltando. O
  `AGENTS.md`, o workflow, a nota da régua e o changelog da raiz param de dizer
  que as referências faltam, e o `AGENTS.md` dá o motivo que vale, com as flags
  de inicialização que levaram o Linux a 1,19% e 1,20% — uma mudança de régua,
  deixada para uma decisão própria.
