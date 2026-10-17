## en

- **The next loop connects the window to a real daemon, and its ruler is written
  before the code.** `npm run check:daemon` builds the app and `dcode` from the
  checkout and runs eight scenarios, each against a fresh `dcode serve` with a
  scripted model: attaching, starting its own, saying why it could not, a new
  session with a conversation, `↵` denying an approval, `1` allowing one,
  steering a running turn, and a daemon that dies. Each is checked against what
  the daemon logged, not only against what the window drew. `docs/loop/` now
  points `/loop` at it, with the ruler protected. It fails on purpose: the window
  does not connect yet, and each scenario says where it stops. Written first
  because a loop that writes its own ruler measures what it built, not what was
  decided.

## pt-BR

- **O próximo loop liga a janela a um daemon de verdade, e a régua dele foi escrita
  antes do código.** O `npm run check:daemon` compila o app e o `dcode` do
  checkout e roda oito cenários, cada um contra um `dcode serve` novo com um
  modelo roteirizado: anexar, subir o próprio, dizer por que não subiu, uma
  sessão nova com conversa, `↵` negando uma aprovação, `1` permitindo, redirecionar
  um turno em andamento, e um daemon que morre. Cada um é conferido contra o que o
  daemon registrou, não só contra o que a janela desenhou. O `docs/loop/` agora
  aponta o `/loop` para ela, com a régua protegida. Reprova de propósito: a janela
  ainda não conecta, e cada cenário diz onde para. Escrita antes porque um loop
  que escreve a própria régua mede o que construiu, não o que foi decidido.
