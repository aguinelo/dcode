## en

- **Six spec families say what the code does.** Documentation only; no
  behaviour changes. `distribution`: the research spec's RN-2 said the
  installer never installs with a warning, while the installer and the
  planning spec have, since August, checked SHA-256 always, covered release
  substitution by the pinned digest or the signature, either one enough, and
  never left anything unverified in silence — RN-2 now says that; RN-1 drops
  the Homebrew tap that left and says `go install` compiles; the planning spec
  stops publishing a Windows artifact no release ever built, and the local
  version text is the Makefile's. `delegated-writing`: no longer "not
  implemented", the delegation floor is the code's 25% and the eval harness
  runs the child; the open question on commands is closed by the invariant
  that already answers it. `learned-memory`: an SDD stability level instead of
  "nothing depends on it" (`GET /v1/memory` does), and the three zero-threshold
  contracts are recorded as never measured. `failure-feedback`: the contract
  it declares and measured at 100% of 20 is no longer called "none", and the
  cut notice is the code's. `done-qualifier`: the config spec's measured
  numbers match the planning spec's, `Measured.Truncated` is documented, the
  review gate is `awaiting_review` and `continue`, a qualifying turn that ends
  without proposing is an explicit failure, and an invariant whose test
  already existed is now claimed. `loop-command`: the daemon conducts and the
  TUI and the desktop follow, in place of the pre-daemon state blocks, and the
  key that stops a loop is `ctrl+c`. Each family has its changelog entry and a
  complete index; two code defects found on the way are reported, not fixed.

## pt-BR

- **Seis famílias de spec dizem o que o código faz.** Só documentação; nenhum
  comportamento muda. `distribution`: a RN-2 do `.r` dizia que o instalador
  nunca instala com aviso, enquanto o instalador e o `.p` fazem, desde agosto,
  SHA-256 sempre, substituição coberta pelo digest fixado ou pela assinatura,
  qualquer uma basta, e nunca não-verificado em silêncio — a RN-2 agora diz
  isso; a RN-1 tira o tap do Homebrew que saiu e diz que o `go install`
  compila; o `.p` deixa de publicar um artefato Windows que nenhum release
  construiu, e o texto de versão local é o do `Makefile`. `delegated-writing`:
  deixa de ser "não implementado", o piso de delegação é o 25% do código e o
  harness de eval roda o filho; a pergunta sobre comandos fecha pela invariante
  que já a respondia. `learned-memory`: um nível de estabilidade do SDD em vez
  de "nada depende" (o `GET /v1/memory` depende), e os três contratos de
  limiar zero ficam registrados como nunca medidos. `failure-feedback`: o
  contrato que ela declara e mediu em 100% de 20 deixa de ser chamado de
  "nenhum", e a marca de corte é a do código. `done-qualifier`: os números
  medidos do `.config` batem com os do `.p`, o `Measured.Truncated` está
  documentado, o portão da revisão é `awaiting_review` e `continue`, o turno
  qualificador que termina sem propor é falha explícita, e uma invariante cujo
  teste já existia passa a ser reivindicada. `loop-command`: o daemon conduz e
  a TUI e o desktop seguem, no lugar dos blocos de estado de antes do daemon, e
  a tecla que para um loop é `ctrl+c`. Cada família tem sua entrada de
  changelog e o índice completo; dois defeitos de código achados no caminho
  ficam relatados, não consertados.
