# A impressão do que foi enviado

**2026-10-13** — a impressão digital que uma medição registra passa a ser a do
prompt que a execução enviou; a tabela de estado conta quantas medições ainda
descrevem o prompt de hoje e quantas descrevem um que mudou.

## O achado

`202609012100-a-medicao-que-nao-sabe-o-que-viu.md` criou `Measurement.Prompt`
para que uma medição dissesse contra qual prompt rodou. O eval preenchia o campo
com `Fixture.Fingerprint()` (`internal/evals/fingerprint.go:21`), que fazia a
soma de `Prompt("")`: formulação em markdown, sem bloco de repositório, sem
memória.

A execução não envia isso. `exchangeRounds` monta cada chamada com
`f.Messages(ctx, p.Family().Name(), w.Dir, history)`
(`internal/evals/contracts_eval_test.go:208`), que passa por `PromptIn`
(`internal/evals/fixture.go:460-476`): a formulação da família — seções com tag
para o claude —, o bloco que diz que não há repositório quando o cenário declara
isso, e a memória do projeto lida pelo leitor do produto.

Ou seja: para qualquer cenário com repositório ausente, memória ou família
claude, a impressão registrada era de um prompt que **nenhum modelo recebeu**.
Das quatro medições que registraram uma, duas são assim —
`floor-yields-to-project` e `floor-names-what-it-yields-to`, cujo cenário declara
repositório ausente.

E nada comparava a impressão com coisa alguma. O comentário de `unverifiable`
dizia "reportado, não imposto"; o que se reportava era só quantas não tinham
impressão, nunca se as que tinham ainda valiam.

## O que muda

- `Fixture.Messages` devolve, junto das mensagens, a impressão do prompt que
  acabou de montar. É a única construção do prompt; a impressão sai dela e não
  de uma segunda montagem ao lado. `Fixture.Fingerprint` deixa de existir.
- O eval registra essa impressão. Se duas rodadas de uma execução, ou duas
  execuções de uma medição, receberem prompts diferentes, a execução **falha em
  medir** — erro, retentativa, e persistindo, medição não confiável. Registrar
  uma das duas seria registrar metade.
- A tabela de estado ganha uma linha contada: das medições que registraram
  impressão, quantas batem com o que o build de hoje envia ao mesmo cenário, na
  família que reivindica o modelo medido, e quantas não batem. Hoje: **1 atual,
  3 velhas**. A guarda reconstrói o prompt pelo mesmo `Messages`, num workspace
  montado como o eval monta.

## O que não muda

- **Nenhuma impressão é inventada.** As dezessete medições sem impressão
  continuam "não registrado". Uma soma calculada hoje é a do prompt de hoje, não
  a do que elas viram.
- **As três impressões velhas ficam como estão.** Duas foram registradas pela
  regra antiga e não descrevem o que foi enviado; a terceira descreve um prompt
  que mudou desde então. Nas três, o que está registrado não bate com hoje, e
  "velha" é a leitura honesta de qualquer um dos dois casos.
- **Continua reportado, não imposto**, pelo mesmo motivo de 1º de setembro: uma
  mudança de prompt que reprovasse o build obrigaria todo PR de prompt a pagar
  cinquenta remedições. O que isto compra é a linha de velhas subir quando o
  prompt muda — visível, com nome.

## O que fica de fora

As definições de ferramenta não entram na impressão. `ce.Assemble` as acrescenta
à mensagem de sistema depois do prompt, e elas vêm do `tools.json` do cenário,
não do produto. Incluí-las agora tornaria incomparável a única impressão que
ainda vale (`boundary-full-access-acts`, registrada pela regra antiga num
cenário em que as duas regras coincidem).

## Invariantes

- `TestTheRecordedFingerprintIsOfThePromptSent`
- `TestAMeasurementWhosePromptMovedIsStale`
- `TestAPromptThatCannotBeRebuiltIsAnError`
- `TestTheStateTableIsCountedAndNotCarried` passa a conferir a linha nova, nas
  duas edições.
