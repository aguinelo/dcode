## en

- **Anthropic requests reuse the prompt cache.** The Claude family marks three
  `cache_control` breakpoints: the last tool, the system prompt and the last
  block of the conversation. With an append-only context, each round now reads
  the previous round's prefix instead of paying full price for it. The decoder
  reads the prompt usage from `message_start`, where the cache numbers
  actually are, and the loop sums cache writes as well as reads, so the cache
  shown in the desktop is no longer always zero. MiniMax M3 on the same dialect
  sends no marker, because nothing says its endpoint reads one.

## pt-BR

- **Requisições Anthropic reaproveitam o cache de prompt.** A família Claude
  marca três pontos `cache_control`: a última ferramenta, o system e o último
  bloco da conversa. Com o contexto append-only, cada rodada lê do cache o
  prefixo da anterior em vez de pagar preço cheio. O decoder lê o uso do
  prompt no `message_start`, onde os números do cache de fato estão, e o laço
  soma o que o cache escreveu além do que leu, então o cache que o desktop
  mostra deixa de estar sempre em zero. O MiniMax M3 no mesmo dialeto não
  manda marca, porque nada diz que o endpoint dele lê uma.
