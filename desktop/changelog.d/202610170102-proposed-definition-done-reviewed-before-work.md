## en

- **A proposed definition of done is reviewed before the work (D56).** While
  a loop waits in `awaiting_review`, its ruler shows a card drawn from
  `done.proposed` as the daemon measured it: the file, each criterion with its
  class (acceptance, regression, broken), what running it did, the command, the
  proposer's reason and the output; the proposer disagreeing with the run, a
  broken criterion and a set with nothing red are warned; protected paths are
  listed. A goal that qualified several folders gets one proposal each.
  "Abrir no editor" asks the main process, which opens only a file the daemon
  listed as that loop's proposal. "Começar o trabalho" continues the loop
  (`POST /v1/loops/{id}/continue`); when another client answered first,
  `loop_not_awaiting_review` is said with the daemon's reason. The Crew's
  Pronto tab gets a Loop section. Before, the review existed only in the TUI.
  Screens 02 and 03 have no loop and stay at 0.00%.

## pt-BR

- **A definição de pronto proposta é revisada antes do trabalho (D56).**
  Enquanto um loop espera em `awaiting_review`, a régua dele mostra um card
  feito do `done.proposed` como o daemon mediu: o arquivo, cada critério com a
  classe (aceitação, regressão, quebrado), o que rodá-lo fez, o comando, o
  porquê do proponente e a saída; a discordância entre o proponente e a
  medição, critério quebrado e conjunto sem nenhum vermelho são avisados; os
  caminhos protegidos aparecem. Um objetivo que qualificou várias pastas ganha
  uma proposta por pasta. "Abrir no editor" pede ao processo principal, que só
  abre um arquivo que o daemon listou como proposta daquele loop. "Começar o
  trabalho" continua o loop (`POST /v1/loops/{id}/continue`); quando outro
  cliente respondeu antes, `loop_not_awaiting_review` aparece com a razão do
  daemon. A aba Pronto da Crew ganha uma seção Loop. Antes, a revisão só
  existia na TUI. As telas 02 e 03 não têm loop e ficam a 0,00%.
