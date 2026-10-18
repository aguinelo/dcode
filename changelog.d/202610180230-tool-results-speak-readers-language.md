## en

- **A finished call's result and the scroll hint speak the reader's
  language, and a dead string naming the wrong key is gone.** The one-line
  result of a tool call (`240 lines (truncated)`, `created, 12 lines`,
  `3 files`, `4 matches in 2 files`, `no matches`, `failed`) and the hint a
  scrolled view shows (`90 lines below · End to follow`) were English literals
  on a pt-BR screen; they now come from the catalogue in both languages. The
  guard against English on a Portuguese screen now puts results and a scrolled
  view on it. `Strings.Interrupt` ("esc interrupts") was read by nothing and
  named `esc` where `ctrl+c` stops a turn; it is removed, and the guard that
  every catalogue string is rendered no longer counts a method call with the
  same name (`Transport.Interrupt`) as a read.

## pt-BR

- **O resultado de uma chamada e a dica de rolagem falam a língua de quem lê,
  e uma frase morta que nomeava a tecla errada saiu.** O resultado de uma linha
  de cada chamada de ferramenta (`240 lines (truncated)`, `created, 12 lines`,
  `3 files`, `4 matches in 2 files`, `no matches`, `failed`) e a dica da vista
  rolada (`90 lines below · End to follow`) eram literais em inglês numa tela
  pt-BR; agora vêm do catálogo, nas duas línguas. A guarda contra inglês numa
  tela em português passa a pôr resultados e uma vista rolada nela.
  `Strings.Interrupt` ("esc interrompe") não era lida por nada e nomeava `esc`
  onde quem para o turno é `ctrl+c`; saiu, e a guarda de que toda frase do
  catálogo é renderizada deixa de contar como leitura uma chamada de método de
  mesmo nome (`Transport.Interrupt`).
