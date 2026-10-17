## en

- **The changes panel opens a PR and shows its CI, always confirmed (D60).**
  Crew's Mudanças tab and the grid's context card show the branch's PR — `#N`
  and title, opening in the browser, its CI as a dot, a word and counts, and
  whether it is open, a draft, merged or closed — or "Criar PR", held with the
  reason on a detached head or on the base itself. "Criar PR" opens a sheet
  prefilled with the conversation's title, its last answer and the files it
  wrote, and the repository's default base, all editable, beside the branch
  that will be pushed; it says what Confirmar does (`git push -u origin`,
  never forced, and the PR through gh; never a merge) and what stays out
  because it is not committed. Nothing is sent before Confirmar. Read when the
  panel opens, on ↻ and on focus at most every 30 s, and every 60 s while CI
  runs and the panel is shown. The owner's request and rule; sidebar badges
  are left out (a GitHub call per row, and they would change the measured
  screen).

## pt-BR

- **O painel de mudanças abre o PR e mostra o CI dele, sempre confirmado
  (D60).** A aba Mudanças da Crew e o cartão de contexto da grade mostram o PR
  da branch — `#N` e o título, que abre no navegador, o CI com ponto, palavra e
  contagens, e se está aberto, em rascunho, merged ou fechado — ou "Criar PR",
  desligado com o motivo em HEAD destacado ou na própria base. "Criar PR" abre
  uma folha preenchida com o título da conversa, a última resposta dela e os
  arquivos que escreveu, e a base padrão do repositório, tudo editável, ao lado
  da branch que será empurrada; diz o que Confirmar faz (`git push -u origin`,
  nunca com força, e o PR pelo gh; nunca merge) e o que fica fora por não estar
  em commit. Nada sai antes de Confirmar. Lido ao abrir o painel, no ↻ e no
  foco no máximo a cada 30 s, e a cada 60 s enquanto o CI roda e o painel está
  à vista. Pedido e regra do dono; o selo na barra lateral fica de fora (uma
  ida ao GitHub por linha, e mudaria a tela medida).
