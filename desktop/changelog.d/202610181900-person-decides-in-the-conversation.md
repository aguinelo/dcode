## en

- **The person decides in the orchestrator's conversation (D67).** The
  orchestrator's question card is drawn in the conversation, where it was
  said, the way Claude Code draws its question card, because the owner wants
  to decide there rather than in a side panel. It shows one item at a time
  with "1 de N" and ‹ ›. The options are radio rows, the selected one checked
  and the recommended one marked. "Outro" takes the person's own words. A
  merge shows the orchestrator's summary and the evidence from the decision
  log. "Pular" and "Próximo ⌘↵" move through the items, and on the last item
  "Confirmar" sends every answer in one `POST /questions/answers`, with the
  skipped items and those never shown in `skipped`. "Aplicar a recomendação a
  todos" sends the recommendations. A proposed card opens on the person's
  words. The result of each answer is said under its item. "◆ N perguntas" in
  the Agentes tab now takes the person to the card, and opens the queue only
  when no card is drawn. The main process checks the batch before it is sent.
  The `orchestra-lists` daemon scenario now answers through the card, with
  the owner's authorization.

## pt-BR

- **A pessoa decide na conversa da orquestradora (D67).** O cartão de
  perguntas da orquestradora aparece na conversa, onde foi dito, do jeito que
  o Claude Code desenha o cartão de pergunta dele, porque o dono quer decidir
  ali e não num painel ao lado. Ele mostra um item por vez, com "1 de N" e
  ‹ ›. As opções são linhas de rádio, a selecionada marcada e a recomendada
  indicada. "Outro" leva as palavras da própria pessoa. Um merge mostra o
  resumo da orquestradora e a evidência do log de decisões. "Pular" e
  "Próximo ⌘↵" passam pelos itens, e no último "Confirmar" manda todas as
  respostas num `POST /questions/answers` só, com os pulados e os não vistos
  em `skipped`. "Aplicar a recomendação a todos" manda as recomendadas. Um
  cartão proposto abre com as palavras da pessoa. O resultado de cada
  resposta aparece embaixo do item. O "◆ N perguntas" da aba Agentes agora
  leva ao cartão, e só abre a fila quando não há cartão desenhado. O processo
  principal confere o lote antes de enviar. O cenário `orchestra-lists` da
  régua do daemon agora responde pelo cartão, com autorização do dono.
