## en

- **A conversation that ends is no longer reported as a failure.** When a
  session closes, the daemon ends its stream and the list shows it ended; the
  window took the stream's end for a fault and said the events had stopped
  coming. It now waits a moment for the list, which arrives over a connection of
  its own, and says so only when the conversation is still live there.

## pt-BR

- **Uma conversa que termina não é mais dada como falha.** Quando uma sessão
  fecha, o daemon encerra o fluxo dela e a lista a mostra terminada; a janela
  tomava o fim do fluxo por defeito e dizia que os eventos tinham parado de chegar.
  Agora espera um instante pela lista, que chega por uma conexão própria, e só
  avisa quando a conversa continua viva nela.
