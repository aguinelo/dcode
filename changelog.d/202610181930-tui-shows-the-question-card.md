## en

- **The TUI shows the question card.** The orchestrator's question card is
  drawn in its transcript where it was said, as text, because the owner wants
  to decide in the conversation, as in Claude Code. The card shows the header
  with "1 de N", the question, the orchestrator's summary and, on a merge, the
  evidence. The options are numbered, with the selected one marked by a
  character and the recommended one said. "Outro" comes last, and a line says
  the card needs the person while items wait. A proposed card opens on the
  person's words. `^Q` puts the keyboard on the card instead of the overlay.
  The arrows move and page, `1`–`9` mark, `s` skips, and `a` applies the
  recommendation to all. Enter goes to the next item, and on the last it
  confirms every answer in one `POST /questions/answers`. Items skipped or
  never shown go in `skipped`, so nothing is answered unseen. A refused answer
  is said with the daemon's code and message. With no card, `^Q` opens the
  oldest question as before.

## pt-BR

- **A TUI mostra o cartão de perguntas.** O cartão de perguntas da
  orquestradora aparece em texto no fluxo dela, onde foi dito, porque o dono
  quer decidir na conversa, como no Claude Code. O cartão mostra a etiqueta
  com "1 de N", a pergunta, o resumo da orquestradora e, num merge, a
  evidência. As opções são numeradas, a selecionada marcada por caractere e a
  recomendada indicada. "Outro" vem por último, e uma linha diz que o cartão
  precisa da pessoa enquanto há item esperando. Um cartão proposto abre com as
  palavras da pessoa. O `^Q` põe o teclado no cartão em vez da sobreposição.
  As setas movem e passam os itens, `1`–`9` marcam, `s` pula e `a` aplica a
  recomendação a todos. Enter vai ao próximo item e, no último, confirma todas
  as respostas num `POST /questions/answers` só. Os pulados e os não vistos vão
  em `skipped`, então nada é respondido sem ser visto. Uma resposta recusada é
  dita com o código e a mensagem do daemon. Sem cartão, o `^Q` abre a pergunta
  mais antiga como antes.
