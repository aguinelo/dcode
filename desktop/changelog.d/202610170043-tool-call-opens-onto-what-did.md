## en

- **A tool call opens onto what it did.** The flow showed each model call as
  one summary line, though `tool.completed` always carried the whole output
  and, for a call that changed a file, its unified diff; only a command typed
  with `!` was shown open. Now a click on the line, or focus and ↵/space,
  opens it: a change shows its diff with + and − coloured as in the git panel
  (the same renderer), anything else its output in monospace with ANSI codes
  stripped, and a failure its whole error. An output past 40 lines opens on
  the first 40 with "mostrar tudo"; a cut by the tool and the daemon's 400-line
  diff cut are said, never silent. Live sessions and recorded histories alike.
  Closed by default and drawn exactly as before: both visual references
  measure 0.00% (D45).

## pt-BR

- **Uma chamada de ferramenta abre no que ela fez.** O fluxo mostrava cada
  chamada do modelo numa linha de resumo, embora o `tool.completed` sempre
  trouxesse a saída inteira e, de quem mudou arquivo, o diff unificado; só um
  comando digitado com `!` aparecia aberto. Agora um clique na linha, ou foco e
  ↵/espaço, abre a chamada: uma mudança mostra o diff com + e − nas cores do
  painel do git (o mesmo desenho), qualquer outra a saída em monoespaçado sem
  os códigos ANSI, e uma falha o erro inteiro. Saída de mais de 40 linhas abre
  nas 40 primeiras, com "mostrar tudo"; o corte da ferramenta e o corte de 400
  linhas do diff no daemon são ditos, nunca calados. Vale para a sessão viva e
  para o histórico gravado. Fechada por padrão e desenhada como antes: as duas
  referências visuais medem 0,00% (D45).
