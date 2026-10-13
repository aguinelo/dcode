# Decisões do desktop — D39 em diante

A continuação de [`DECISIONS-2.md`](DECISIONS-2.md), que vai de D21 a D40, e de
[`DECISIONS.md`](DECISIONS.md), que tem a introdução, D1 a D20, as decisões em
aberto, as lacunas e os pedidos ao núcleo. A numeração é uma só; D39 veio para cá
porque o DECISIONS-2 passaria de 500 linhas com ela.

**D39. A aprovação oferece as respostas que a TUI oferece** (emenda D22). A
janela oferecia sempre uma vez, nesta sessão e negar; o protocolo tem cinco
respostas, e a TUI escolhe quais pela travessia (`approvalKeys`,
`standingScope`). Agora a janela faz o mesmo:

- **Rede:** permitir uma vez, neste projeto, sempre, e negar — sem "nesta
  sessão", como na TUI. Só a rede tem resposta que sobrevive à sessão: uma
  concessão permanente de escrita fora do workspace seria o sistema de arquivos
  inteiro, dado uma vez por um motivo que ninguém registra.
- **Qualquer outra travessia:** permitir uma vez, nesta sessão, e negar.
- **Teclas:** de `1` a `n` na ordem mostrada; `↵` e `esc` negam, e negar continua
  a opção destacada (D22). As respostas que ficam gravadas — neste projeto,
  sempre — vêm no tom de aviso, como na TUI, com o título dizendo onde ficam.
- **A razão do daemon aparece no cartão**, abaixo do que se pede, e não só no
  título: é o que se lê antes de responder, e um título só aparece a quem passa o
  ponteiro.

A tela de referência 03 é uma travessia de rede, então ela muda com isto: quatro
opções no lugar de três, e a razão numa linha. A captura nova é decisão de quem
aprova o visual, num PR próprio, como `tests/visual/states.json` pede.

Descartados: manter as três para todas (a janela ofereceria "nesta sessão" para
a rede, que a TUI não oferece, e esconderia as duas respostas permanentes);
oferecer as cinco sempre (concessão permanente de escrita fora do projeto).
